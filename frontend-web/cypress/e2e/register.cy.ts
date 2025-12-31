describe("Register Page", () => {
  beforeEach(() => {
    cy.visit("/register");
  });

  describe("UI Elements", () => {
    it("should display register form", () => {
      cy.getByTestId("register-form").should("be.visible");
    });

    it("should display all input fields", () => {
      cy.getByTestId("register-name-input").should("be.visible");
      cy.getByTestId("register-email-input").should("be.visible");
      cy.getByTestId("register-password-input").should("be.visible");
      cy.getByTestId("register-confirm-password-input").should("be.visible");
    });

    it("should display submit button", () => {
      cy.getByTestId("register-submit-button")
        .should("be.visible")
        .and("contain.text", "Create Account");
    });

    it("should display login link", () => {
      cy.getByTestId("register-login-link")
        .should("be.visible")
        .and("have.attr", "href", "/login");
    });
  });

  describe("Form Validation", () => {
    it("should show error for empty name", () => {
      cy.getByTestId("register-email-input").type("test@example.com");
      cy.getByTestId("register-password-input").type("Test123!");
      cy.getByTestId("register-confirm-password-input").type("Test123!");
      cy.getByTestId("register-submit-button").click();
      cy.getByTestId("register-name-error").should("be.visible");
    });

    it("should show error for short name", () => {
      cy.getByTestId("register-name-input").type("A");
      cy.getByTestId("register-name-input").blur();
      cy.getByTestId("register-name-error")
        .should("be.visible")
        .and("contain.text", "at least 2 characters");
    });

    it("should show error for invalid email", () => {
      cy.getByTestId("register-email-input").type("invalidemail");
      cy.getByTestId("register-email-input").blur();
      cy.getByTestId("register-email-error")
        .should("be.visible")
        .and("contain.text", "invalid");
    });

    it("should show error for weak password", () => {
      cy.getByTestId("register-password-input").type("weak");
      cy.getByTestId("register-password-input").blur();
      cy.getByTestId("register-password-error").should("be.visible");
    });

    it("should show error for password without uppercase", () => {
      cy.getByTestId("register-password-input").type("testtest1!");
      cy.getByTestId("register-password-input").blur();
      cy.getByTestId("register-password-error")
        .should("be.visible")
        .and("contain.text", "uppercase");
    });

    it("should show error for password without number", () => {
      cy.getByTestId("register-password-input").type("TestTest!");
      cy.getByTestId("register-password-input").blur();
      cy.getByTestId("register-password-error")
        .should("be.visible")
        .and("contain.text", "number");
    });

    it("should show error for password without special character", () => {
      cy.getByTestId("register-password-input").type("TestTest1");
      cy.getByTestId("register-password-input").blur();
      cy.getByTestId("register-password-error")
        .should("be.visible")
        .and("contain.text", "special character");
    });

    it("should show error for mismatched passwords", () => {
      cy.getByTestId("register-password-input").type("Test123!");
      cy.getByTestId("register-confirm-password-input").type("Test456!");
      cy.getByTestId("register-confirm-password-input").blur();
      cy.getByTestId("register-confirm-password-error")
        .should("be.visible")
        .and("contain.text", "do not match");
    });
  });

  describe("Password Toggle", () => {
    it("should toggle password visibility", () => {
      cy.getByTestId("register-password-input").type("Test123!");

      // Initially password should be hidden
      cy.getByTestId("register-password-input").should("have.attr", "type", "password");

      // Click toggle button
      cy.getByTestId("register-toggle-password").click();

      // Password should now be visible
      cy.getByTestId("register-password-input").should("have.attr", "type", "text");
    });

    it("should toggle confirm password visibility", () => {
      cy.getByTestId("register-confirm-password-input").type("Test123!");

      // Initially password should be hidden
      cy.getByTestId("register-confirm-password-input").should("have.attr", "type", "password");

      // Click toggle button
      cy.getByTestId("register-toggle-confirm-password").click();

      // Password should now be visible
      cy.getByTestId("register-confirm-password-input").should("have.attr", "type", "text");
    });
  });

  describe("Navigation", () => {
    it("should navigate to login page", () => {
      cy.getByTestId("register-login-link").click();
      cy.url().should("include", "/login");
    });
  });

  describe("Form Submission", () => {
    it("should show loading state when submitting", () => {
      // Fill in valid form data
      cy.getByTestId("register-name-input").type("Test User");
      cy.getByTestId("register-email-input").type("newuser@example.com");
      cy.getByTestId("register-password-input").type("Test123!");
      cy.getByTestId("register-confirm-password-input").type("Test123!");

      // Intercept API call to delay response
      cy.intercept("POST", "**/auth/register", (req) => {
        req.reply({
          delay: 1000,
          statusCode: 201,
          body: { message: "User created" }
        });
      }).as("registerRequest");

      cy.getByTestId("register-submit-button").click();

      // Button should show loading state
      cy.getByTestId("register-submit-button")
        .should("be.disabled")
        .and("contain.text", "Creating Account");
    });

    it("should show error message on failed registration", () => {
      // Intercept API call to return error
      cy.intercept("POST", "**/auth/register", {
        statusCode: 409,
        body: { message: "Email already exists" }
      }).as("registerRequest");

      cy.getByTestId("register-name-input").type("Test User");
      cy.getByTestId("register-email-input").type("existing@example.com");
      cy.getByTestId("register-password-input").type("Test123!");
      cy.getByTestId("register-confirm-password-input").type("Test123!");
      cy.getByTestId("register-submit-button").click();

      cy.wait("@registerRequest");

      // Error message should appear
      cy.getByTestId("register-error").should("be.visible");
    });
  });
});
