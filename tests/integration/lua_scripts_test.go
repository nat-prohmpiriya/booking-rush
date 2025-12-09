package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// Test Lua scripts for reserve and release seats

const reserveSeatsScript = `--[[
    Reserve Seats Lua Script
--]]

local zone_availability_key = KEYS[1]
local user_reservations_key = KEYS[2]
local reservation_key = KEYS[3]

local quantity = tonumber(ARGV[1])
local max_per_user = tonumber(ARGV[2])
local user_id = ARGV[3]
local booking_id = ARGV[4]
local zone_id = ARGV[5]
local event_id = ARGV[6]
local show_id = ARGV[7]
local unit_price = ARGV[8]
local ttl_seconds = tonumber(ARGV[9]) or 600

if not quantity or quantity <= 0 then
    return {0, "INVALID_QUANTITY", "Quantity must be a positive number"}
end

local available = redis.call("GET", zone_availability_key)
if not available then
    return {0, "ZONE_NOT_FOUND", "Zone availability not initialized"}
end
available = tonumber(available)

if available < quantity then
    return {0, "INSUFFICIENT_STOCK", "Not enough seats available. Available: " .. available .. ", Requested: " .. quantity}
end

local user_reserved = redis.call("GET", user_reservations_key)
user_reserved = tonumber(user_reserved) or 0

if max_per_user and max_per_user > 0 then
    if (user_reserved + quantity) > max_per_user then
        return {0, "USER_LIMIT_EXCEEDED", "User limit exceeded. Current: " .. user_reserved .. ", Requested: " .. quantity .. ", Max: " .. max_per_user}
    end
end

local remaining = redis.call("DECRBY", zone_availability_key, quantity)
local new_user_reserved = redis.call("INCRBY", user_reservations_key, quantity)
redis.call("EXPIRE", user_reservations_key, ttl_seconds + 60)

local timestamp = redis.call("TIME")
local created_at = timestamp[1] .. "." .. timestamp[2]

redis.call("HSET", reservation_key,
    "booking_id", booking_id,
    "user_id", user_id,
    "zone_id", zone_id,
    "event_id", event_id,
    "show_id", show_id,
    "quantity", quantity,
    "unit_price", unit_price,
    "status", "reserved",
    "created_at", created_at,
    "expires_at", timestamp[1] + ttl_seconds
)

redis.call("EXPIRE", reservation_key, ttl_seconds)

return {1, remaining, new_user_reserved}
`

const releaseSeatsScript = `--[[
    Release Seats Lua Script
--]]

local zone_availability_key = KEYS[1]
local user_reservations_key = KEYS[2]
local reservation_key = KEYS[3]

local booking_id = ARGV[1]
local user_id = ARGV[2]

local reservation = redis.call("HGETALL", reservation_key)
if #reservation == 0 then
    return {0, "RESERVATION_NOT_FOUND", "Reservation does not exist or has expired"}
end

local reservation_data = {}
for i = 1, #reservation, 2 do
    reservation_data[reservation[i]] = reservation[i + 1]
end

if reservation_data["booking_id"] ~= booking_id then
    return {0, "INVALID_BOOKING_ID", "Booking ID does not match"}
end

if reservation_data["user_id"] ~= user_id then
    return {0, "INVALID_USER_ID", "User ID does not match"}
end

local status = reservation_data["status"]
if status ~= "reserved" then
    return {0, "ALREADY_RELEASED", "Reservation status is '" .. (status or "unknown") .. "', cannot release"}
end

local quantity = tonumber(reservation_data["quantity"])
if not quantity or quantity <= 0 then
    return {0, "INVALID_QUANTITY", "Invalid quantity in reservation"}
end

local new_available = redis.call("INCRBY", zone_availability_key, quantity)

local current_user_reserved = redis.call("GET", user_reservations_key)
current_user_reserved = tonumber(current_user_reserved) or 0

local new_user_reserved = current_user_reserved - quantity
if new_user_reserved < 0 then
    new_user_reserved = 0
end

if new_user_reserved > 0 then
    redis.call("SET", user_reservations_key, new_user_reserved)
    redis.call("EXPIRE", user_reservations_key, 660)
else
    redis.call("DEL", user_reservations_key)
end

redis.call("DEL", reservation_key)

return {1, new_available, new_user_reserved}
`

func getTestRedisConfig() *redis.Config {
	cfg := redis.DefaultConfig()

	if host := os.Getenv("TEST_REDIS_HOST"); host != "" {
		cfg.Host = host
	}
	if password := os.Getenv("TEST_REDIS_PASSWORD"); password != "" {
		cfg.Password = password
	}

	return cfg
}

func TestReleaseSeats_Success(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	// Test data
	zoneID := "test-zone-release-001"
	userID := "test-user-release-001"
	eventID := "test-event-release-001"
	bookingID := "test-booking-release-001"

	zoneKey := "zone:availability:" + zoneID
	userKey := "user:reservations:" + userID + ":" + eventID
	reservationKey := "reservation:" + bookingID

	// Cleanup
	defer client.Del(ctx, zoneKey, userKey, reservationKey)

	// Setup: Initialize zone with 100 seats
	client.Set(ctx, zoneKey, "100", time.Hour)

	// Step 1: Reserve 2 seats
	_, err = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	if err != nil {
		t.Fatalf("Failed to load reserve script: %v", err)
	}

	result, err := client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey},
		2, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", 600,
	).Slice()
	if err != nil {
		t.Fatalf("Reserve failed: %v", err)
	}

	success := result[0].(int64)
	if success != 1 {
		t.Fatalf("Reserve should succeed, got: %v", result)
	}

	remainingAfterReserve := result[1].(int64)
	if remainingAfterReserve != 98 {
		t.Errorf("Expected 98 remaining after reserve, got %d", remainingAfterReserve)
	}

	// Step 2: Release the seats
	_, err = client.LoadScript(ctx, "release_seats", releaseSeatsScript)
	if err != nil {
		t.Fatalf("Failed to load release script: %v", err)
	}

	result, err = client.EvalShaByName(ctx, "release_seats",
		[]string{zoneKey, userKey, reservationKey},
		bookingID, userID,
	).Slice()
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	success = result[0].(int64)
	if success != 1 {
		t.Fatalf("Release should succeed, got: %v", result)
	}

	newAvailable := result[1].(int64)
	if newAvailable != 100 {
		t.Errorf("Expected 100 available after release, got %d", newAvailable)
	}

	newUserReserved := result[2].(int64)
	if newUserReserved != 0 {
		t.Errorf("Expected 0 user reserved after release, got %d", newUserReserved)
	}

	// Verify reservation is deleted
	exists, _ := client.Exists(ctx, reservationKey).Result()
	if exists != 0 {
		t.Error("Reservation key should be deleted after release")
	}

	// Verify user reservation key is deleted (was 0)
	exists, _ = client.Exists(ctx, userKey).Result()
	if exists != 0 {
		t.Error("User reservation key should be deleted when count is 0")
	}
}

func TestReleaseSeats_ReservationNotFound(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	// Test data - no reservation exists
	zoneKey := "zone:availability:test-zone-notfound"
	userKey := "user:reservations:test-user-notfound:test-event-notfound"
	reservationKey := "reservation:test-booking-notfound"

	// Cleanup
	defer client.Del(ctx, zoneKey, userKey, reservationKey)

	client.Set(ctx, zoneKey, "100", time.Hour)

	_, err = client.LoadScript(ctx, "release_seats", releaseSeatsScript)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}

	result, err := client.EvalShaByName(ctx, "release_seats",
		[]string{zoneKey, userKey, reservationKey},
		"booking-xxx", "user-xxx",
	).Slice()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	success := result[0].(int64)
	if success != 0 {
		t.Errorf("Expected failure (0), got %d", success)
	}

	errorCode := result[1].(string)
	if errorCode != "RESERVATION_NOT_FOUND" {
		t.Errorf("Expected error code RESERVATION_NOT_FOUND, got %s", errorCode)
	}
}

func TestReleaseSeats_InvalidBookingID(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	zoneID := "test-zone-invalidbooking"
	userID := "test-user-invalidbooking"
	eventID := "test-event-invalidbooking"
	bookingID := "test-booking-invalidbooking"

	zoneKey := "zone:availability:" + zoneID
	userKey := "user:reservations:" + userID + ":" + eventID
	reservationKey := "reservation:" + bookingID

	defer client.Del(ctx, zoneKey, userKey, reservationKey)

	client.Set(ctx, zoneKey, "100", time.Hour)

	// Reserve first
	_, _ = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	_, _ = client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey},
		2, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", 600,
	).Slice()

	// Try to release with wrong booking ID
	_, _ = client.LoadScript(ctx, "release_seats", releaseSeatsScript)
	result, err := client.EvalShaByName(ctx, "release_seats",
		[]string{zoneKey, userKey, reservationKey},
		"wrong-booking-id", userID,
	).Slice()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	success := result[0].(int64)
	if success != 0 {
		t.Errorf("Expected failure (0), got %d", success)
	}

	errorCode := result[1].(string)
	if errorCode != "INVALID_BOOKING_ID" {
		t.Errorf("Expected error code INVALID_BOOKING_ID, got %s", errorCode)
	}
}

func TestReleaseSeats_InvalidUserID(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	zoneID := "test-zone-invaliduser"
	userID := "test-user-invaliduser"
	eventID := "test-event-invaliduser"
	bookingID := "test-booking-invaliduser"

	zoneKey := "zone:availability:" + zoneID
	userKey := "user:reservations:" + userID + ":" + eventID
	reservationKey := "reservation:" + bookingID

	defer client.Del(ctx, zoneKey, userKey, reservationKey)

	client.Set(ctx, zoneKey, "100", time.Hour)

	// Reserve first
	_, _ = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	_, _ = client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey},
		2, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", 600,
	).Slice()

	// Try to release with wrong user ID
	_, _ = client.LoadScript(ctx, "release_seats", releaseSeatsScript)
	result, err := client.EvalShaByName(ctx, "release_seats",
		[]string{zoneKey, userKey, reservationKey},
		bookingID, "wrong-user-id",
	).Slice()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	success := result[0].(int64)
	if success != 0 {
		t.Errorf("Expected failure (0), got %d", success)
	}

	errorCode := result[1].(string)
	if errorCode != "INVALID_USER_ID" {
		t.Errorf("Expected error code INVALID_USER_ID, got %s", errorCode)
	}
}

func TestReleaseSeats_PartialUserReservations(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	zoneID := "test-zone-partial"
	userID := "test-user-partial"
	eventID := "test-event-partial"
	bookingID1 := "test-booking-partial-001"
	bookingID2 := "test-booking-partial-002"

	zoneKey := "zone:availability:" + zoneID
	userKey := "user:reservations:" + userID + ":" + eventID
	reservationKey1 := "reservation:" + bookingID1
	reservationKey2 := "reservation:" + bookingID2

	defer client.Del(ctx, zoneKey, userKey, reservationKey1, reservationKey2)

	// Initialize with 100 seats
	client.Set(ctx, zoneKey, "100", time.Hour)

	_, _ = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	_, _ = client.LoadScript(ctx, "release_seats", releaseSeatsScript)

	// Reserve 3 seats in booking 1
	result1, _ := client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey1},
		3, 10, userID, bookingID1, zoneID, eventID, "show-001", "1000", 600,
	).Slice()
	if result1[0].(int64) != 1 {
		t.Fatalf("First reservation should succeed")
	}

	// Reserve 2 more seats in booking 2 (total user reserved: 5)
	result2, _ := client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey2},
		2, 10, userID, bookingID2, zoneID, eventID, "show-001", "1000", 600,
	).Slice()
	if result2[0].(int64) != 1 {
		t.Fatalf("Second reservation should succeed")
	}

	// Verify 95 remaining seats
	remaining, _ := client.Get(ctx, zoneKey).Int64()
	if remaining != 95 {
		t.Errorf("Expected 95 remaining, got %d", remaining)
	}

	// Release booking 1 (3 seats)
	releaseResult, err := client.EvalShaByName(ctx, "release_seats",
		[]string{zoneKey, userKey, reservationKey1},
		bookingID1, userID,
	).Slice()
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	if releaseResult[0].(int64) != 1 {
		t.Fatalf("Release should succeed, got: %v", releaseResult)
	}

	newAvailable := releaseResult[1].(int64)
	if newAvailable != 98 {
		t.Errorf("Expected 98 available after releasing 3 seats, got %d", newAvailable)
	}

	newUserReserved := releaseResult[2].(int64)
	if newUserReserved != 2 {
		t.Errorf("Expected 2 user reserved (booking 2 still exists), got %d", newUserReserved)
	}

	// Verify user key still exists with value 2
	userReserved, _ := client.Get(ctx, userKey).Int64()
	if userReserved != 2 {
		t.Errorf("User reservation key should have value 2, got %d", userReserved)
	}
}

func TestReleaseSeats_AtomicityUnderConcurrency(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	zoneID := "test-zone-atomic"
	eventID := "test-event-atomic"
	zoneKey := "zone:availability:" + zoneID

	// Initialize with 1000 seats
	client.Set(ctx, zoneKey, "1000", time.Hour)

	_, _ = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	_, _ = client.LoadScript(ctx, "release_seats", releaseSeatsScript)

	// Create 10 reservations
	type reservation struct {
		userID         string
		bookingID      string
		userKey        string
		reservationKey string
		quantity       int
	}

	reservations := make([]reservation, 10)
	keysToCleanup := []string{zoneKey}

	for i := 0; i < 10; i++ {
		r := reservation{
			userID:    "user-atomic-" + string(rune('A'+i)),
			bookingID: "booking-atomic-" + string(rune('A'+i)),
			quantity:  (i % 4) + 1, // 1-4 seats each
		}
		r.userKey = "user:reservations:" + r.userID + ":" + eventID
		r.reservationKey = "reservation:" + r.bookingID
		reservations[i] = r
		keysToCleanup = append(keysToCleanup, r.userKey, r.reservationKey)
	}
	defer client.Del(ctx, keysToCleanup...)

	// Reserve all
	totalReserved := 0
	for _, r := range reservations {
		result, _ := client.EvalShaByName(ctx, "reserve_seats",
			[]string{zoneKey, r.userKey, r.reservationKey},
			r.quantity, 10, r.userID, r.bookingID, zoneID, eventID, "show-001", "1000", 600,
		).Slice()
		if result[0].(int64) == 1 {
			totalReserved += r.quantity
		}
	}

	remaining, _ := client.Get(ctx, zoneKey).Int64()
	expected := 1000 - int64(totalReserved)
	if remaining != expected {
		t.Errorf("Expected %d remaining after reservations, got %d", expected, remaining)
	}

	// Release all concurrently
	done := make(chan bool, 10)
	for _, r := range reservations {
		go func(r reservation) {
			_, _ = client.EvalShaByName(ctx, "release_seats",
				[]string{zoneKey, r.userKey, r.reservationKey},
				r.bookingID, r.userID,
			).Slice()
			done <- true
		}(r)
	}

	// Wait for all releases
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final check: should be back to 1000
	finalAvailable, _ := client.Get(ctx, zoneKey).Int64()
	if finalAvailable != 1000 {
		t.Errorf("Expected 1000 available after all releases, got %d", finalAvailable)
	}
}
