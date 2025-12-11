#!/bin/bash

# Seed Events Script
# Creates 10 events with shows and zones via API
# Mixed statuses: OPEN, UPCOMING, and ENDED events

set -e

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@bookingrush.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin123@}"
ADMIN_NAME="${ADMIN_NAME:-Admin User}"

# Database configuration (for creating admin user)
DB_HOST="${DB_HOST:-100.104.0.42}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-booking_user}"
DB_PASSWORD="${DB_PASSWORD:-booking_password}"
DB_NAME="${DB_NAME:-booking_db}"

echo "=== Event Seed Script ==="
echo "API URL: $API_BASE_URL"
echo "Today: $(date +%Y-%m-%d)"
echo ""
echo "Event Status Distribution:"
echo "  - 5 events: OPEN FOR SALE"
echo "  - 3 events: UPCOMING (not yet on sale)"
echo "  - 2 events: ENDED (past events)"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Step 0: Try to register admin user (if not exists)
echo -e "${YELLOW}[Step 0] Checking/Creating admin user...${NC}"
REGISTER_RESPONSE=$(curl -s -X POST "$API_BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\", \"name\": \"$ADMIN_NAME\"}")

REG_SUCCESS=$(echo $REGISTER_RESPONSE | jq -r '.success // false')
if [ "$REG_SUCCESS" = "true" ]; then
  echo -e "${GREEN}Admin user created!${NC}"

  # Update user role to organizer in database
  echo -e "${YELLOW}Updating user role to organizer...${NC}"
  PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c \
    "UPDATE users SET role = 'organizer' WHERE email = '$ADMIN_EMAIL';" 2>/dev/null || {
    echo -e "${YELLOW}Could not update role via psql. Please update manually:${NC}"
    echo "UPDATE users SET role = 'organizer' WHERE email = '$ADMIN_EMAIL';"
  }
else
  USER_EXISTS=$(echo $REGISTER_RESPONSE | jq -r '.code // empty')
  if [ "$USER_EXISTS" = "USER_EXISTS" ]; then
    echo -e "${GREEN}Admin user already exists.${NC}"
  else
    echo -e "${YELLOW}Registration response: $(echo $REGISTER_RESPONSE | jq -r '.message // .error // "unknown"')${NC}"
  fi
fi
echo ""

# Step 1: Login to get JWT token
echo -e "${YELLOW}[Step 1] Logging in as admin...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\"}")

# Extract access token
ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.access_token // empty')

if [ -z "$ACCESS_TOKEN" ]; then
  echo -e "${RED}Failed to login. Response:${NC}"
  echo $LOGIN_RESPONSE | jq .
  echo ""
  echo -e "${YELLOW}Troubleshooting:${NC}"
  echo "1. Make sure auth service is running"
  echo "2. Try registering manually:"
  echo "   curl -X POST $API_BASE_URL/auth/register -H 'Content-Type: application/json' \\"
  echo "     -d '{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\", \"name\": \"$ADMIN_NAME\"}'"
  echo ""
  echo "3. Update user role to organizer in database:"
  echo "   UPDATE users SET role = 'organizer' WHERE email = '$ADMIN_EMAIL';"
  exit 1
fi

echo -e "${GREEN}Login successful!${NC}"
echo ""

# Function to create an event
create_event() {
  local name="$1"
  local description="$2"
  local short_description="$3"
  local venue_name="$4"
  local venue_address="$5"
  local city="$6"
  local poster_url="$7"
  local banner_url="$8"
  local booking_start="$9"
  local booking_end="${10}"
  local max_tickets="${11:-4}"

  RESPONSE=$(curl -s -X POST "$API_BASE_URL/events" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -d "{
      \"name\": \"$name\",
      \"description\": \"$description\",
      \"short_description\": \"$short_description\",
      \"venue_name\": \"$venue_name\",
      \"venue_address\": \"$venue_address\",
      \"city\": \"$city\",
      \"country\": \"Thailand\",
      \"poster_url\": \"$poster_url\",
      \"banner_url\": \"$banner_url\",
      \"max_tickets_per_user\": $max_tickets,
      \"booking_start_at\": \"$booking_start\",
      \"booking_end_at\": \"$booking_end\",
      \"meta_title\": \"$name - Book Now\",
      \"meta_description\": \"$short_description\"
    }")

  EVENT_ID=$(echo $RESPONSE | jq -r '.data.id // empty')

  if [ -z "$EVENT_ID" ]; then
    echo -e "${RED}    Failed to create event${NC}"
    echo $RESPONSE | jq .
    return 1
  fi

  echo -e "${GREEN}    Event created: $EVENT_ID${NC}"
  echo $EVENT_ID
}

# Function to create a show for an event
create_show() {
  local event_id="$1"
  local name="$2"
  local show_date="$3"
  local start_time="$4"
  local end_time="$5"

  RESPONSE=$(curl -s -X POST "$API_BASE_URL/events/$event_id/shows" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -d "{
      \"name\": \"$name\",
      \"show_date\": \"$show_date\",
      \"start_time\": \"$start_time\",
      \"end_time\": \"$end_time\"
    }")

  SHOW_ID=$(echo $RESPONSE | jq -r '.data.id // empty')

  if [ -z "$SHOW_ID" ]; then
    echo -e "${RED}    Failed to create show${NC}"
    echo $RESPONSE | jq .
    return 1
  fi

  echo -e "${GREEN}    Show: $name ($show_date)${NC}"
  echo $SHOW_ID
}

# Function to create a zone for a show
create_zone() {
  local show_id="$1"
  local name="$2"
  local price="$3"
  local total_seats="$4"
  local description="$5"

  RESPONSE=$(curl -s -X POST "$API_BASE_URL/shows/$show_id/zones" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -d "{
      \"name\": \"$name\",
      \"price\": $price,
      \"total_seats\": $total_seats,
      \"description\": \"$description\",
      \"sort_order\": 0
    }")

  ZONE_ID=$(echo $RESPONSE | jq -r '.data.id // empty')

  if [ -z "$ZONE_ID" ]; then
    echo -e "${RED}    Failed to create zone${NC}"
    echo $RESPONSE | jq .
    return 1
  fi
}

# Function to publish an event
publish_event() {
  local event_id="$1"

  RESPONSE=$(curl -s -X POST "$API_BASE_URL/events/$event_id/publish" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

  SUCCESS=$(echo $RESPONSE | jq -r '.success // false')

  if [ "$SUCCESS" != "true" ]; then
    echo -e "${RED}    Failed to publish${NC}"
    echo $RESPONSE | jq .
    return 1
  fi

  echo -e "${GREEN}    Published!${NC}"
}

echo "=== Creating 10 Events ==="
echo ""

# Today: 2025-12-10

# ============================================
# EVENT 1: OPEN - BTS Concert (Show: Dec 31, 2025)
# ============================================
echo -e "${GREEN}[1/10] OPEN${NC} - BTS World Tour: Love Yourself"
EVENT1_ID=$(create_event \
  "BTS World Tour: Love Yourself" \
  "Experience the global phenomenon BTS live in Bangkok! Join millions of ARMYs for an unforgettable night of music, dance, and connection. The Love Yourself World Tour brings the biggest K-pop group to Thailand for two spectacular nights." \
  "BTS Live in Bangkok - World Tour 2025" \
  "Rajamangala National Stadium" \
  "286 Ramkhamhaeng Rd, Hua Mak" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1540039155733-5bb30b53aa14?w=800" \
  "https://images.unsplash.com/photo-1459749411175-04bf5292ceea?w=1600" \
  "2025-12-01T10:00:00+07:00" \
  "2025-12-30T23:59:59+07:00" \
  4)

if [ ! -z "$EVENT1_ID" ]; then
  SHOW1_ID=$(create_show "$EVENT1_ID" "Night 1" "2025-12-31" "19:00:00+07:00" "23:00:00+07:00")
  if [ ! -z "$SHOW1_ID" ]; then
    create_zone "$SHOW1_ID" "VVIP Standing" 12000 500 "Front stage standing area with exclusive merchandise"
    create_zone "$SHOW1_ID" "VIP Standing" 8500 1000 "Premium standing area with great view"
    create_zone "$SHOW1_ID" "Gold Seated" 5500 2000 "Reserved seating in lower bowl"
    create_zone "$SHOW1_ID" "Silver Seated" 3500 3000 "Reserved seating in upper bowl"
  fi
  SHOW2_ID=$(create_show "$EVENT1_ID" "Night 2" "2026-01-01" "19:00:00+07:00" "23:00:00+07:00")
  if [ ! -z "$SHOW2_ID" ]; then
    create_zone "$SHOW2_ID" "VVIP Standing" 12000 500 "Front stage standing area with exclusive merchandise"
    create_zone "$SHOW2_ID" "VIP Standing" 8500 1000 "Premium standing area with great view"
    create_zone "$SHOW2_ID" "Gold Seated" 5500 2000 "Reserved seating in lower bowl"
    create_zone "$SHOW2_ID" "Silver Seated" 3500 3000 "Reserved seating in upper bowl"
  fi
  publish_event "$EVENT1_ID"
fi
echo ""

# ============================================
# EVENT 2: UPCOMING - Ed Sheeran (Sale starts Dec 20)
# ============================================
echo -e "${BLUE}[2/10] UPCOMING${NC} - Ed Sheeran Mathematics Tour"
EVENT2_ID=$(create_event \
  "Ed Sheeran Mathematics Tour" \
  "Grammy Award winner Ed Sheeran brings his Mathematics Tour to Bangkok. Experience acoustic perfection as Ed performs his greatest hits including Shape of You, Perfect, and Bad Habits in an intimate stadium setting." \
  "Ed Sheeran Live - Mathematics Tour Bangkok" \
  "Impact Arena" \
  "99 Popular Rd, Pak Kret" \
  "Nonthaburi" \
  "https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=800" \
  "https://images.unsplash.com/photo-1501386761578-eac5c94b800a?w=1600" \
  "2025-12-20T10:00:00+07:00" \
  "2026-01-28T23:59:59+07:00" \
  4)

if [ ! -z "$EVENT2_ID" ]; then
  SHOW_ID=$(create_show "$EVENT2_ID" "Bangkok Show" "2026-01-29" "20:00:00+07:00" "23:00:00+07:00")
  if [ ! -z "$SHOW_ID" ]; then
    create_zone "$SHOW_ID" "CAT 1" 8900 800 "Best seats in the house - center stage view"
    create_zone "$SHOW_ID" "CAT 2" 6500 1500 "Premium side stage seating"
    create_zone "$SHOW_ID" "CAT 3" 4500 2000 "Great value seating with full stage view"
    create_zone "$SHOW_ID" "CAT 4" 2500 2500 "Budget-friendly seating"
  fi
  publish_event "$EVENT2_ID"
fi
echo ""

# ============================================
# EVENT 3: OPEN - Coldplay (Show: Feb 14, 2026)
# ============================================
echo -e "${GREEN}[3/10] OPEN${NC} - Coldplay Music of the Spheres"
EVENT3_ID=$(create_event \
  "Coldplay Music of the Spheres" \
  "Coldplay returns to Bangkok with their spectacular Music of the Spheres World Tour. Featuring LED wristbands for all attendees, biodegradable confetti, and an eco-friendly production that has redefined live music." \
  "Coldplay World Tour 2026 - Bangkok" \
  "Rajamangala National Stadium" \
  "286 Ramkhamhaeng Rd, Hua Mak" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1470229722913-7c0e2dbbafd3?w=800" \
  "https://images.unsplash.com/photo-1429962714451-bb934ecdc4ec?w=1600" \
  "2025-11-15T10:00:00+07:00" \
  "2026-02-13T23:59:59+07:00" \
  4)

if [ ! -z "$EVENT3_ID" ]; then
  SHOW_ID=$(create_show "$EVENT3_ID" "Bangkok Show" "2026-02-14" "19:30:00+07:00" "22:30:00+07:00")
  if [ ! -z "$SHOW_ID" ]; then
    create_zone "$SHOW_ID" "Infinity Ticket" 9500 1000 "GA Standing - closest to stage"
    create_zone "$SHOW_ID" "A Reserve" 7500 2000 "Premium reserved seating"
    create_zone "$SHOW_ID" "B Reserve" 5500 3000 "Standard reserved seating"
    create_zone "$SHOW_ID" "C Reserve" 3500 4000 "Value seating"
  fi
  publish_event "$EVENT3_ID"
fi
echo ""

# ============================================
# EVENT 4: ENDED - Summer Music Festival (Show: Nov 15, 2025)
# ============================================
echo -e "${RED}[4/10] ENDED${NC} - Summer Sonic Bangkok"
EVENT4_ID=$(create_event \
  "Summer Sonic Bangkok" \
  "Thailand's premier summer music festival featuring top international and local artists across multiple stages. A celebration of rock, pop, and electronic music." \
  "Summer Sonic Festival 2025 - Bangkok Edition" \
  "BITEC Bangna" \
  "88 Bangna-Trad Road" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1533174072545-7a4b6ad7a6c3?w=800" \
  "https://images.unsplash.com/photo-1506157786151-b8491531f063?w=1600" \
  "2025-10-01T10:00:00+07:00" \
  "2025-11-14T23:59:59+07:00" \
  4)

if [ ! -z "$EVENT4_ID" ]; then
  SHOW_ID=$(create_show "$EVENT4_ID" "Festival Day" "2025-11-15" "12:00:00+07:00" "23:00:00+07:00")
  if [ ! -z "$SHOW_ID" ]; then
    create_zone "$SHOW_ID" "VIP" 5500 500 "VIP viewing area with amenities"
    create_zone "$SHOW_ID" "General Admission" 2500 5000 "Full festival access"
    create_zone "$SHOW_ID" "Early Bird" 1990 1000 "Limited early bird pricing"
  fi
  publish_event "$EVENT4_ID"
fi
echo ""

# ============================================
# EVENT 5: OPEN - Muay Thai Championship (Show: Jan 9, 2026)
# ============================================
echo -e "${GREEN}[5/10] OPEN${NC} - Muay Thai Super Fight Night"
EVENT5_ID=$(create_event \
  "Muay Thai Super Fight Night" \
  "Witness the best Muay Thai fighters from around the world compete in this championship event. Featuring 10 world-class bouts including the WBC Muay Thai World Championship main event." \
  "World Championship Muay Thai - Bangkok" \
  "Lumpinee Boxing Stadium" \
  "6 Ramintra Rd, Anusawari" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1549719386-74dfcbf7dbed?w=800" \
  "https://images.unsplash.com/photo-1544367567-0f2fcb009e0b?w=1600" \
  "2025-11-01T10:00:00+07:00" \
  "2026-01-08T23:59:59+07:00" \
  6)

if [ ! -z "$EVENT5_ID" ]; then
  SHOW_ID=$(create_show "$EVENT5_ID" "Championship Night" "2026-01-09" "18:00:00+07:00" "23:00:00+07:00")
  if [ ! -z "$SHOW_ID" ]; then
    create_zone "$SHOW_ID" "Ringside" 5000 100 "Ringside seats - feel every punch"
    create_zone "$SHOW_ID" "VIP" 3000 200 "VIP seating with complimentary drinks"
    create_zone "$SHOW_ID" "Standard" 1500 500 "Standard seating with great view"
    create_zone "$SHOW_ID" "Standing" 800 300 "Standing area"
  fi
  publish_event "$EVENT5_ID"
fi
echo ""

# ============================================
# EVENT 6: UPCOMING - Jazz Festival (Sale starts Dec 25)
# ============================================
echo -e "${BLUE}[6/10] UPCOMING${NC} - Bangkok International Jazz Festival"
EVENT6_ID=$(create_event \
  "Bangkok International Jazz Festival" \
  "Three days of world-class jazz featuring international and local artists. From smooth jazz to fusion, experience the best of jazz music under the stars at Lumpini Park." \
  "Bangkok Jazz Festival 2026 - 3 Day Pass Available" \
  "Lumpini Park" \
  "Rama IV Road" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1415201364774-f6f0bb35f28f?w=800" \
  "https://images.unsplash.com/photo-1511192336575-5a79af67a629?w=1600" \
  "2025-12-25T10:00:00+07:00" \
  "2026-01-18T23:59:59+07:00" \
  4)

if [ ! -z "$EVENT6_ID" ]; then
  SHOW1_ID=$(create_show "$EVENT6_ID" "Day 1 - Opening Night" "2026-01-19" "17:00:00+07:00" "23:00:00+07:00")
  if [ ! -z "$SHOW1_ID" ]; then
    create_zone "$SHOW1_ID" "VIP Table" 3500 50 "Reserved table for 4 with bottle service"
    create_zone "$SHOW1_ID" "Premium GA" 1800 500 "Premium standing/seating area"
    create_zone "$SHOW1_ID" "General Admission" 900 1000 "Standard festival access"
  fi
  SHOW2_ID=$(create_show "$EVENT6_ID" "Day 2 - Main Event" "2026-01-20" "17:00:00+07:00" "23:00:00+07:00")
  if [ ! -z "$SHOW2_ID" ]; then
    create_zone "$SHOW2_ID" "VIP Table" 3500 50 "Reserved table for 4 with bottle service"
    create_zone "$SHOW2_ID" "Premium GA" 1800 500 "Premium standing/seating area"
    create_zone "$SHOW2_ID" "General Admission" 900 1000 "Standard festival access"
  fi
  publish_event "$EVENT6_ID"
fi
echo ""

# ============================================
# EVENT 7: OPEN - Symphony Orchestra (Show: Dec 30, 2025)
# ============================================
echo -e "${GREEN}[7/10] OPEN${NC} - Royal Bangkok Symphony Orchestra"
EVENT7_ID=$(create_event \
  "Royal Bangkok Symphony Orchestra" \
  "An evening of classical masterpieces performed by the Royal Bangkok Symphony Orchestra. Program includes Beethoven's Symphony No. 9, Tchaikovsky's Piano Concerto No. 1, and works by Thai composers." \
  "Classical Night - Beethoven and Tchaikovsky" \
  "Thailand Cultural Centre" \
  "14 Thiam Ruam Mit Rd, Huai Khwang" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1465847899084-d164df4dedc6?w=800" \
  "https://images.unsplash.com/photo-1507838153414-b4b713384a76?w=1600" \
  "2025-11-01T10:00:00+07:00" \
  "2025-12-29T23:59:59+07:00" \
  4)

if [ ! -z "$EVENT7_ID" ]; then
  SHOW_ID=$(create_show "$EVENT7_ID" "Evening Performance" "2025-12-30" "19:30:00+07:00" "22:00:00+07:00")
  if [ ! -z "$SHOW_ID" ]; then
    create_zone "$SHOW_ID" "Orchestra" 2500 200 "Best acoustic experience"
    create_zone "$SHOW_ID" "Mezzanine" 1800 300 "Elevated view of the orchestra"
    create_zone "$SHOW_ID" "Balcony" 1200 400 "Upper level seating"
  fi
  publish_event "$EVENT7_ID"
fi
echo ""

# ============================================
# EVENT 8: ENDED - Comedy Night (Show: Nov 28, 2025)
# ============================================
echo -e "${RED}[8/10] ENDED${NC} - Stand-up Comedy Night"
EVENT8_ID=$(create_event \
  "Stand-up Comedy Night: Thai Edition" \
  "A night of laughter featuring Thailand's top comedians. From observational humor to political satire, this show brings together the best of Thai comedy talent." \
  "Comedy Night Bangkok - Thai Comedians Special" \
  "Scala Theater" \
  "Siam Square Soi 1" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1585699324551-f6c309eedeca?w=800" \
  "https://images.unsplash.com/photo-1527224538127-2104bb71c51b?w=1600" \
  "2025-11-01T10:00:00+07:00" \
  "2025-11-27T23:59:59+07:00" \
  4)

if [ ! -z "$EVENT8_ID" ]; then
  SHOW_ID=$(create_show "$EVENT8_ID" "Evening Show" "2025-11-28" "20:00:00+07:00" "22:30:00+07:00")
  if [ ! -z "$SHOW_ID" ]; then
    create_zone "$SHOW_ID" "VIP" 1500 100 "Front row seating with meet and greet"
    create_zone "$SHOW_ID" "Standard" 800 300 "Standard seating"
    create_zone "$SHOW_ID" "Economy" 500 200 "Back section seating"
  fi
  publish_event "$EVENT8_ID"
fi
echo ""

# ============================================
# EVENT 9: OPEN - Street Food Festival (Show: Jan 11, 2026)
# ============================================
echo -e "${GREEN}[9/10] OPEN${NC} - Bangkok Street Food Festival"
EVENT9_ID=$(create_event \
  "Bangkok Street Food Festival" \
  "Celebrate Thailand's culinary heritage at this 3-day food festival. Over 100 vendors, cooking demonstrations by celebrity chefs, and unlimited food sampling with VIP passes." \
  "Street Food Festival - Taste of Thailand" \
  "Central World Square" \
  "999/9 Rama I Rd" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1555939594-58d7cb561ad1?w=800" \
  "https://images.unsplash.com/photo-1504674900247-0877df9cc836?w=1600" \
  "2025-11-01T10:00:00+07:00" \
  "2026-01-10T23:59:59+07:00" \
  6)

if [ ! -z "$EVENT9_ID" ]; then
  SHOW_ID=$(create_show "$EVENT9_ID" "Weekend Pass" "2026-01-11" "11:00:00+07:00" "22:00:00+07:00")
  if [ ! -z "$SHOW_ID" ]; then
    create_zone "$SHOW_ID" "VIP All-You-Can-Eat" 1500 500 "Unlimited food sampling + priority access"
    create_zone "$SHOW_ID" "Premium Pass" 800 1000 "10 food vouchers included"
    create_zone "$SHOW_ID" "General Entry" 299 2000 "Festival entry only"
  fi
  publish_event "$EVENT9_ID"
fi
echo ""

# ============================================
# EVENT 10: UPCOMING - Tech Conference (Sale starts Jan 1, 2026)
# ============================================
echo -e "${BLUE}[10/10] UPCOMING${NC} - TechCrunch Bangkok 2026"
EVENT10_ID=$(create_event \
  "TechCrunch Bangkok 2026" \
  "Southeast Asia's premier technology conference. Featuring keynotes from tech leaders, startup pitch competitions, networking events, and the latest in AI, blockchain, and fintech innovations." \
  "TechCrunch Conference - Innovation Summit" \
  "Queen Sirikit National Convention Center" \
  "60 New Rachadapisek Rd" \
  "Bangkok" \
  "https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=800" \
  "https://images.unsplash.com/photo-1505373877841-8d25f7d46678?w=1600" \
  "2026-01-01T10:00:00+07:00" \
  "2026-02-17T23:59:59+07:00" \
  2)

if [ ! -z "$EVENT10_ID" ]; then
  SHOW1_ID=$(create_show "$EVENT10_ID" "Day 1 - Keynotes" "2026-02-18" "09:00:00+07:00" "18:00:00+07:00")
  if [ ! -z "$SHOW1_ID" ]; then
    create_zone "$SHOW1_ID" "VIP All-Access" 15000 100 "All sessions + networking dinner + swag bag"
    create_zone "$SHOW1_ID" "Conference Pass" 8500 500 "All sessions access"
    create_zone "$SHOW1_ID" "Startup Pass" 3500 300 "Discounted rate for startups"
    create_zone "$SHOW1_ID" "Student Pass" 1500 200 "Student discount with valid ID"
  fi
  SHOW2_ID=$(create_show "$EVENT10_ID" "Day 2 - Workshops" "2026-02-19" "09:00:00+07:00" "18:00:00+07:00")
  if [ ! -z "$SHOW2_ID" ]; then
    create_zone "$SHOW2_ID" "VIP All-Access" 15000 100 "All sessions + networking dinner + swag bag"
    create_zone "$SHOW2_ID" "Conference Pass" 8500 500 "All sessions access"
    create_zone "$SHOW2_ID" "Startup Pass" 3500 300 "Discounted rate for startups"
    create_zone "$SHOW2_ID" "Student Pass" 1500 200 "Student discount with valid ID"
  fi
  publish_event "$EVENT10_ID"
fi
echo ""

echo "=== Seed Complete ==="
echo ""
echo -e "${GREEN}Successfully created 10 events!${NC}"
echo ""
echo "Summary:"
echo -e "  ${GREEN}OPEN (5):${NC} BTS, Coldplay, Muay Thai, Symphony, Food Festival"
echo -e "  ${BLUE}UPCOMING (3):${NC} Ed Sheeran, Jazz Festival, TechCrunch"
echo -e "  ${RED}ENDED (2):${NC} Summer Sonic, Comedy Night"
echo ""
echo "View events at: http://localhost:3000/events"
