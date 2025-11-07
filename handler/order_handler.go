package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dispatchsvc "github.com/mikios34/delivery-backend/dispatch"
	orderpkg "github.com/mikios34/delivery-backend/order"
)

type OrderHandler struct {
	service  orderpkg.Service
	dispatch dispatchsvc.Service
}

func NewOrderHandler(svc orderpkg.Service, d dispatchsvc.Service) *OrderHandler {
	return &OrderHandler{service: svc, dispatch: d}
}

type createOrderPayload struct {
	CustomerID          string   `json:"customer_id" binding:"required"`
	TypeID              string   `json:"type_id" binding:"required"`
	VehicleTypeID       string   `json:"vehicle_type_id" binding:"required"`
	ReceiverPhone       string   `json:"receiver_phone" binding:"required"`
	PickupAddress       string   `json:"pickup_address" binding:"required"`
	PickupLat           *float64 `json:"pickup_lat"`
	PickupLng           *float64 `json:"pickup_lng"`
	DropoffAddress      string   `json:"dropoff_address" binding:"required"`
	DropoffLat          *float64 `json:"dropoff_lat"`
	DropoffLng          *float64 `json:"dropoff_lng"`
	EstimatedPriceCents int64    `json:"estimated_price_cents" binding:"required"`
}

func (h *OrderHandler) CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p createOrderPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		cid, err := uuid.Parse(p.CustomerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
			return
		}
		tid, err := uuid.Parse(p.TypeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type_id"})
			return
		}
		vtid, err := uuid.Parse(p.VehicleTypeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vehicle_type_id"})
			return
		}
		req := orderpkg.CreateOrderRequest{
			CustomerID:          cid,
			TypeID:              tid,
			VehicleTypeID:       vtid,
			ReceiverPhone:       p.ReceiverPhone,
			PickupAddress:       p.PickupAddress,
			PickupLat:           p.PickupLat,
			PickupLng:           p.PickupLng,
			DropoffAddress:      p.DropoffAddress,
			DropoffLat:          p.DropoffLat,
			DropoffLng:          p.DropoffLng,
			EstimatedPriceCents: p.EstimatedPriceCents,
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		created, err := h.service.CreateOrder(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order", "detail": err.Error()})
			return
		}
		// auto-dispatch synchronously for now
		assignedOrder, assignedCourier, derr := h.dispatch.FindAndAssign(ctx, created.ID)
		if derr != nil {
			// return created order without assignment but include error info
			c.JSON(http.StatusCreated, gin.H{"order": created, "dispatch_error": derr.Error()})
			return
		}
		if assignedCourier == nil {
			c.JSON(http.StatusCreated, gin.H{"order": assignedOrder, "message": "no available couriers"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"order": assignedOrder, "assigned_courier_id": assignedCourier.ID})
	}
}

func (h *OrderHandler) ListOrderTypes() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		types, err := h.service.ListOrderTypes(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, types)
	}
}

// EstimateTariffs estimates delivery tariffs for all active vehicle types based on pickup/dropoff coordinates.
// GET /api/v1/orders/tariffs?pickup_lat=&pickup_lng=&dropoff_lat=&dropoff_lng=
func (h *OrderHandler) EstimateTariffs(repo orderpkg.Repository) gin.HandlerFunc {
	type tariffResp struct {
		VehicleTypeID string  `json:"vehicle_type_id"`
		Code          string  `json:"code"`
		Name          string  `json:"name"`
		DistanceKm    float64 `json:"distance_km"`
		DurationMin   float64 `json:"duration_min"`
		Price         float64 `json:"price"`
		PriceCents    int64   `json:"price_cents"`
	}

	haversineKm := func(lat1, lon1, lat2, lon2 float64) float64 {
		// Haversine formula in km
		const R = 6371.0 // Earth radius in km
		toRad := func(d float64) float64 { return d * (math.Pi / 180.0) }
		dLat := toRad(lat2 - lat1)
		dLon := toRad(lon2 - lon1)
		a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
		c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))
		return R * c
	}

	// map vehicle code to OSRM profile
	profileFor := func(code string) string {
		code = strings.ToLower(strings.TrimSpace(code))
		switch code {
		case "bike", "bicycle", "cycle":
			return "cycling"
		case "walker", "walk", "foot":
			return "walking"
		default:
			// motorbike, motor, scooter, car, taxi, other -> driving
			return "driving"
		}
	}

	type osrmResponse struct {
		Routes []struct {
			Distance float64 `json:"distance"` // meters
			Duration float64 `json:"duration"` // seconds
		} `json:"routes"`
		Code string `json:"code"`
		Msg  string `json:"message"`
	}

	getRoute := func(ctx context.Context, baseURL, profile string, oLat, oLng, dLat, dLng float64) (float64, float64, error) {
		url := fmt.Sprintf("%s/route/v1/%s/%.6f,%.6f;%.6f,%.6f?overview=false&alternatives=false&steps=false", strings.TrimRight(baseURL, "/"), profile, oLng, oLat, dLng, dLat)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return 0, 0, fmt.Errorf("routing status %d", resp.StatusCode)
		}
		var rr osrmResponse
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&rr); err != nil {
			return 0, 0, err
		}
		if rr.Code != "Ok" || len(rr.Routes) == 0 {
			return 0, 0, fmt.Errorf("routing error: %s", rr.Msg)
		}
		distKm := rr.Routes[0].Distance / 1000.0
		durMin := rr.Routes[0].Duration / 60.0
		return distKm, durMin, nil
	}

	return func(c *gin.Context) {
		// Parse query params
		q := c.Request.URL.Query()
		pLatStr := q.Get("pickup_lat")
		pLngStr := q.Get("pickup_lng")
		dLatStr := q.Get("dropoff_lat")
		dLngStr := q.Get("dropoff_lng")
		if pLatStr == "" || pLngStr == "" || dLatStr == "" || dLngStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pickup_lat, pickup_lng, dropoff_lat, dropoff_lng are required query params"})
			return
		}
		pLat, err1 := strconv.ParseFloat(pLatStr, 64)
		pLng, err2 := strconv.ParseFloat(pLngStr, 64)
		dLat, err3 := strconv.ParseFloat(dLatStr, 64)
		dLng, err4 := strconv.ParseFloat(dLngStr, 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lat/lng values"})
			return
		}
		// Basic bounds check
		if pLat < -90 || pLat > 90 || dLat < -90 || dLat > 90 || pLng < -180 || pLng > 180 || dLng < -180 || dLng > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lat must be [-90,90], lng must be [-180,180]"})
			return
		}

		distKmFallback := haversineKm(pLat, pLng, dLat, dLng)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		// Pull active vehicle types with pricing
		types, err := repo.ListActiveVehicleTypes(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vehicle types", "detail": err.Error()})
			return
		}
		// Prepare routing per unique profile to avoid duplicate calls
		baseURL := os.Getenv("OSRM_BASE_URL")
		if baseURL == "" {
			baseURL = "https://router.project-osrm.org"
		}
		profileSet := map[string]struct{}{}
		for _, vt := range types {
			profileSet[profileFor(vt.Code)] = struct{}{}
		}
		routeByProfile := map[string]struct{ dist, dur float64 }{}
		for profile := range profileSet {
			if profile == "" {
				continue
			}
			dist, dur, err := getRoute(ctx, baseURL, profile, pLat, pLng, dLat, dLng)
			if err != nil {
				// record zero to signal fallback
				routeByProfile[profile] = struct{ dist, dur float64 }{0, 0}
				continue
			}
			routeByProfile[profile] = struct{ dist, dur float64 }{dist, dur}
		}

		out := make([]tariffResp, 0, len(types))
		for _, vt := range types {
			prof := profileFor(vt.Code)
			r, ok := routeByProfile[prof]
			distKm := distKmFallback
			durMin := 0.0
			if ok && r.dist > 0 && r.dur > 0 {
				distKm = r.dist
				durMin = r.dur
			} else {
				// fallback duration from average speed
				speed := vt.AvgSpeedKmh
				if speed <= 0 {
					speed = 30
				}
				durMin = (distKm / speed) * 60
			}
			// price = max(minimum, base + per_km*distance + per_minute*duration + booking_fee)
			calc := vt.BaseFare + vt.PerKm*distKm + vt.PerMinute*durMin + vt.BookingFee
			if calc < vt.MinimumFare {
				calc = vt.MinimumFare
			}
			priceRounded := math.Round(calc*100) / 100
			priceCents := int64(math.Round(calc * 100))
			out = append(out, tariffResp{
				VehicleTypeID: vt.ID.String(),
				Code:          vt.Code,
				Name:          vt.Name,
				DistanceKm:    math.Round(distKm*100) / 100,
				DurationMin:   math.Round(durMin*10) / 10,
				Price:         priceRounded,
				PriceCents:    priceCents,
			})
		}
		c.JSON(http.StatusOK, gin.H{"tariffs": out})
	}
}
