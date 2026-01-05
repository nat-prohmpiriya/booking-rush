describe("Home Page", () => {
  beforeEach(() => {
    cy.visit("/");
  });

  describe("UI Elements", () => {
    it("should display home page", () => {
      cy.getByTestId("home-page").should("be.visible");
    });

    it("should display header", () => {
      cy.getByTestId("header").should("be.visible");
    });

    it("should display events container", () => {
      cy.getByTestId("home-events-container").should("be.visible");
    });
  });

  describe("Header Navigation", () => {
    it("should display header logo", () => {
      cy.getByTestId("header-logo").should("be.visible");
    });

    it("should navigate to login when clicking login button", () => {
      cy.getByTestId("header-login-button").click();
      cy.url().should("include", "/login");
    });
  });

  describe("Event Sections", () => {
    it("should display event sections when events exist", () => {
      // Mock API response with events
      cy.intercept("GET", "**/events*", {
        statusCode: 200,
        body: {
          data: [
            {
              id: "test-event-1",
              name: "Test Concert",
              venue: "Test Venue",
              image_url: "/test-image.jpg",
              min_price: 500,
              sale_status: "on_sale",
              shows: [
                {
                  id: "show-1",
                  show_date: "2025-01-15",
                  sale_status: "on_sale"
                }
              ]
            }
          ]
        }
      }).as("getEvents");

      cy.visit("/");
      cy.wait("@getEvents");

      cy.getByTestId("event-section-on-sale-now").should("exist");
    });

    it("should display no events message when no events", () => {
      // Mock API response with empty events
      cy.intercept("GET", "**/events*", {
        statusCode: 200,
        body: {
          data: []
        }
      }).as("getEvents");

      cy.visit("/");
      cy.wait("@getEvents");

      cy.getByTestId("home-no-events").should("be.visible");
    });
  });

  describe("Event Card Interaction", () => {
    beforeEach(() => {
      // Mock API response with events
      cy.intercept("GET", "**/events*", {
        statusCode: 200,
        body: {
          data: [
            {
              id: "event-123",
              name: "Amazing Concert",
              venue: "Grand Hall",
              image_url: "/concert.jpg",
              min_price: 1500,
              sale_status: "on_sale",
              shows: [
                {
                  id: "show-1",
                  show_date: "2025-02-20",
                  sale_status: "on_sale"
                }
              ]
            }
          ]
        }
      }).as("getEvents");

      cy.visit("/");
      cy.wait("@getEvents");
    });

    it("should display event card with correct information", () => {
      cy.getByTestId("event-card-event-123").should("be.visible");
      cy.getByTestId("event-card-title").first().should("contain.text", "Amazing Concert");
      cy.getByTestId("event-card-venue").first().should("contain.text", "Grand Hall");
    });

    it("should navigate to event detail when clicking book button", () => {
      cy.getByTestId("event-card-book-button").first().click();
      cy.url().should("include", "/events/event-123");
    });
  });

  describe("Responsive Design", () => {
    it("should show mobile menu button on small screens", () => {
      cy.viewport("iphone-x");
      cy.getByTestId("header-mobile-menu-button").should("be.visible");
    });

    it("should show desktop navigation on large screens", () => {
      cy.viewport(1280, 720);
      cy.getByTestId("header-desktop-nav").should("be.visible");
    });
  });

  describe("Loading State", () => {
    it("should show loading skeleton while fetching events", () => {
      // Intercept with delay to see loading state
      cy.intercept("GET", "**/events*", (req) => {
        req.reply({
          delay: 2000,
          statusCode: 200,
          body: { data: [] }
        });
      }).as("getEvents");

      cy.visit("/");

      // Should show loading state initially
      cy.get(".animate-pulse").should("exist");
    });
  });
});
