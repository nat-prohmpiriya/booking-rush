describe("Login Page", () => {
  beforeEach(() => {
    cy.visit("/login");
  });

  describe("UI Elements", () => {
    it("should display login form", () => {
      cy.getByTestId("login-form").should("be.visible");
    });

    it("should display email input", () => {
      cy.getByTestId("login-email-input").should("be.visible");
    });

    it("should display password input", () => {
      cy.getByTestId("login-password-input").should("be.visible");
    });

    it("should display submit button", () => {
      cy.getByTestId("login-submit-button")
        .should("be.visible")
        .and("contain.text", "Sign In");
    });

    it("should display remember me checkbox", () => {
      cy.getByTestId("login-remember-checkbox").should("be.visible");
    });

    it("should display forgot password link", () => {
      cy.getByTestId("login-forgot-password-link")
        .should("be.visible")
        .and("have.attr", "href", "/forgot-password");
    });

    it("should display register link", () => {
      cy.getByTestId("login-register-link")
        .should("be.visible")
        .and("have.attr", "href", "/register");
    });
  });

  describe("Form Validation", () => {
    it("should show error for empty email", () => {
      cy.getByTestId("login-password-input").type("password123");
      cy.getByTestId("login-submit-button").click();
      cy.getByTestId("login-email-error").should("be.visible");
    });

    it("should show error for invalid email format", () => {
      cy.getByTestId("login-email-input").type("invalidemail");
      cy.getByTestId("login-email-input").blur();
      cy.getByTestId("login-email-error")
        .should("be.visible")
        .and("contain.text", "invalid");
    });

    it("should show error for empty password", () => {
      cy.getByTestId("login-email-input").type("test@example.com");
      cy.getByTestId("login-submit-button").click();
      cy.getByTestId("login-password-error").should("be.visible");
    });

    it("should show error for short password", () => {
      cy.getByTestId("login-password-input").type("12345");
      cy.getByTestId("login-password-input").blur();
      cy.getByTestId("login-password-error")
        .should("be.visible")
        .and("contain.text", "at least 6 characters");
    });
  });

  describe("Password Toggle", () => {
    it("should toggle password visibility", () => {
      cy.getByTestId("login-password-input").type("password123");

      // Initially password should be hidden
      cy.getByTestId("login-password-input").should("have.attr", "type", "password");

      // Click toggle button
      cy.getByTestId("login-toggle-password").click();

      // Password should now be visible
      cy.getByTestId("login-password-input").should("have.attr", "type", "text");

      // Click toggle again
      cy.getByTestId("login-toggle-password").click();

      // Password should be hidden again
      cy.getByTestId("login-password-input").should("have.attr", "type", "password");
    });
  });

  describe("Navigation", () => {
    it("should navigate to register page", () => {
      cy.getByTestId("login-register-link").click();
      cy.url().should("include", "/register");
    });

    it("should navigate to forgot password page", () => {
      cy.getByTestId("login-forgot-password-link").click();
      cy.url().should("include", "/forgot-password");
    });
  });

  describe("Form Submission", () => {
    it("should disable button while loading", () => {
      // Fill in valid credentials
      cy.getByTestId("login-email-input").type("test@example.com");
      cy.getByTestId("login-password-input").type("Test123!");

      // Intercept API call to delay response
      cy.intercept("POST", "**/auth/login", (req) => {
        req.reply({
          delay: 1000,
          statusCode: 200,
          body: { token: "fake-token" }
        });
      }).as("loginRequest");

      cy.getByTestId("login-submit-button").click();

      // Button should show loading state
      cy.getByTestId("login-submit-button")
        .should("be.disabled")
        .and("contain.text", "Signing in");
    });

    it("should show error message on failed login", () => {
      // Intercept API call to return error
      cy.intercept("POST", "**/auth/login", {
        statusCode: 401,
        body: { message: "Invalid credentials" }
      }).as("loginRequest");

      cy.getByTestId("login-email-input").type("test@example.com");
      cy.getByTestId("login-password-input").type("wrongpassword");
      cy.getByTestId("login-submit-button").click();

      cy.wait("@loginRequest");

      // Error message should appear
      cy.getByTestId("login-error").should("be.visible");
    });
  });
});
