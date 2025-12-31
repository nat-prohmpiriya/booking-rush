// ***********************************************
// Custom commands for Cypress tests
// ***********************************************

/**
 * Get element by data-testid attribute
 * Usage: cy.getByTestId('login-form')
 */
Cypress.Commands.add("getByTestId", (testId: string) => {
  return cy.get(`[data-testid="${testId}"]`);
});

/**
 * Login via UI
 * Usage: cy.login('test@example.com', 'password123')
 */
Cypress.Commands.add("login", (email: string, password: string) => {
  cy.visit("/login");
  cy.getByTestId("login-email-input").type(email);
  cy.getByTestId("login-password-input").type(password);
  cy.getByTestId("login-submit-button").click();
});
