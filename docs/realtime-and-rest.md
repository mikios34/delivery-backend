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
  - Creates an order. Dispatch runs immediately: if a courier is found, status becomes "assigned"; otherwise it becomes "no_nearby_driver".

- POST /api/v1/courier/availability
  - Auth: courier
  - Body: { available: boolean }
  - Reads courier_id from token; no courier_id in body.

- POST /api/v1/courier/orders/{accept|decline|arrived|picked|delivered}
  - Auth: courier
  - Body: { order_id: string, courier_id: string }
  - On accepted/picked_up/delivered the customer receives "order.status" enriched with courier_name/phone/profile_picture.

- GET /api/v1/customer/active-order (alias: /activeOrder)
  - Auth: customer
  - 200 OK -> { active: false } when none
  - 200 OK -> { active: true, order: Order, assigned_driver?: { id, name, phone, profile_picture? } }

- GET /api/v1/courier/active-order (alias: /activeOrder)
  - Auth: courier
  - 200 OK -> { active: false } when none
  - 200 OK -> { active: true, order: Order }

## Notes

- Active orders are those with status NOT IN (no_nearby_driver, delivered).
- A background job reassigns orders stuck in "assigned" every 15s with a 15s cutoff and avoids retrying the same courier.
- WebSocket writes are serialized per-connection to prevent concurrent write races.
