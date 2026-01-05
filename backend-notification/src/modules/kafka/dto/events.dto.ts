/**
 * Kafka event types for notifications
 */
export enum EventType {
  PAYMENT_SUCCESS = 'payment.success',
  PAYMENT_FAILED = 'payment.failed',
  BOOKING_EXPIRED = 'booking.expired',
  BOOKING_CANCELLED = 'booking.cancelled',
  BOOKING_CONFIRMED = 'booking.confirmed',
}

/**
 * Base event structure
 */
export interface BaseEvent {
  event_type: string;
  timestamp: string;
  correlation_id?: string;
}

/**
 * Payment success event from payment service
 */
export interface PaymentSuccessEvent extends BaseEvent {
  event_type: 'payment.success';
  booking_id: string;
  tenant_id: string;
  user_id: string;
  user_email: string;
  payment_id: string;
  amount: number;
  currency: string;
  payment_method: string;
  // Booking details
  event_id: string;
  event_name: string;
  show_id: string;
  show_date: string;
  zone_id: string;
  zone_name: string;
  quantity: number;
  unit_price: number;
  confirmation_code: string;
  venue_name?: string;
  venue_address?: string;
}

/**
 * Booking expired event
 */
export interface BookingExpiredEvent extends BaseEvent {
  event_type: 'booking.expired';
  booking_id: string;
  tenant_id: string;
  user_id: string;
  user_email: string;
  event_id: string;
  event_name: string;
  show_date: string;
  zone_name: string;
  quantity: number;
  expired_at: string;
}

/**
 * Booking cancelled event
 */
export interface BookingCancelledEvent extends BaseEvent {
  event_type: 'booking.cancelled';
  booking_id: string;
  tenant_id: string;
  user_id: string;
  user_email: string;
  event_id: string;
  event_name: string;
  confirmation_code: string;
  cancelled_at: string;
  reason?: string;
  refund_amount?: number;
}

/**
 * Union type for all events
 */
export type BookingEvent =
  | PaymentSuccessEvent
  | BookingExpiredEvent
  | BookingCancelledEvent;
