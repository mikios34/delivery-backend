# Realtime events and REST contracts

This document summarizes the WebSocket events and REST endpoints relevant to dispatch, status updates, and active orders.

## WebSocket

Base paths (authenticated):
- Courier WS: GET /api/v1/ws/courier (role=courier)
- Customer WS: GET /api/v1/ws/customer (role=customer)

Message envelope:
- { "event": string, "data": any }

### Courier events

- event: "order.assigned"
  - data: OrderAssignedPayload
  - { order_id, customer_id, pickup_address, pickup_lat?, pickup_lng?, dropoff_address, dropoff_lat?, dropoff_lng?, receiver_phone }

- event: "order.assignment_timed_out"
  - data: { order_id, customer_id }

- event: "order.reassigned_away"
  - data: { order_id, customer_id }

- event: "order.no_nearby_driver"
  - data: { order_id, customer_id }

### Customer events

- event: "order.status"
  - data: OrderStatusPayload
  - Fields:
    - order_id: string
    - status: "assigned" | "accepted" | "declined" | "arrived" | "picked_up" | "delivered" | "no_nearby_driver"
    - When status == "assigned": pickup_address?, pickup_lat?, pickup_lng?, dropoff_address?, dropoff_lat?, dropoff_lng?, receiver_phone?
    - When status in [accepted, picked_up, delivered]: courier_name?, courier_phone?, courier_profile_picture?

## REST endpoints

- POST /api/v1/orders
  - Auth: customer
  - Body:
    - pickup_address: string
    - pickup_lat: number
    - pickup_lng: number
    - dropoff_address: string
    - dropoff_lat: number
    - dropoff_lng: number
    - receiver_phone: string
    - order_type_id?: string (UUID) — if your app distinguishes order categories
    - vehicle_type_id: string (UUID) — selected from GET /api/v1/orders/tariffs
    - estimated_price_cents: number — price returned by the tariffs endpoint for the selected vehicle type
  - 200 OK -> Order
  - Notes:
    - Clients should first call GET /api/v1/orders/tariffs to retrieve pricing per vehicle type and then post the chosen vehicle_type_id and estimated_price_cents here.
    - Server currently stores the client-sent estimated_price_cents. If you need server-side verification, consider recalculating on create and rejecting mismatches beyond a small tolerance.

  - Auth: customer
  - Creates an order. Dispatch runs immediately: if a courier is found, status becomes "assigned"; otherwise it becomes "no_nearby_driver".

  - Auth: courier
  - Body: { available: boolean }
  - Reads courier_id from token; no courier_id in body.

  - Auth: courier
  - Body: { first_name, last_name, phone, has_vehicle?, primary_vehicle?, vehicle_details?, guaranty_option_id, firebase_uid, profile_picture? }
  - On accepted/picked_up/delivered the customer receives "order.status" enriched with courier_name/phone/profile_picture.

- GET /api/v1/customer/active-order (alias: /activeOrder)
  - Auth: customer
  - 200 OK -> { active: false } when none
  - 200 OK -> { active: true, order: Order, assigned_driver?: { id, name, phone, profile_picture? } }

- GET /api/v1/courier/active-order (alias: /activeOrder)
  - Auth: courier
  - 200 OK -> { active: false } when none
  - 200 OK -> { active: true, order: Order }

- GET /api/v1/orders/tariffs
  - Auth: customer
  - Query: pickup_lat, pickup_lng, dropoff_lat, dropoff_lng (all required)
  - 200 OK -> { tariffs: [ { vehicle_type_id, code, name, distance_km, duration_min, price, price_cents } ] }
  - Notes:
    - Distance and duration are computed via OSRM (profile chosen by vehicle type: cycling/walking/driving). Fallback to Haversine + average speed when routing fails.
    - price = max(minimum_fare, base_fare + per_km*distance_km + per_minute*duration_min + booking_fee)
    - Prefer using price_cents when creating an order to avoid floating-point rounding issues.

## Notes

- Active orders are those with status NOT IN (no_nearby_driver, delivered).
- A background job reassigns orders stuck in "assigned" every 15s with a 15s cutoff and avoids retrying the same courier.
- WebSocket writes are serialized per-connection to prevent concurrent write races.
