// ***********************************************************
// This file is processed and loaded automatically before your test files.
// You can read more here: https://on.cypress.io/configuration
// ***********************************************************

import "./commands";

// Prevent TypeScript errors when accessing Cypress
declare global {
  namespace Cypress {
    interface Chainable {
      /**
       * Custom command to get element by data-testid
       * @example cy.getByTestId('login-form')
       */
      getByTestId(testId: string): Chainable<JQuery<HTMLElement>>;

      /**
       * Custom command to login via API
       * @example cy.login('test@example.com', 'password123')
       */
      login(email: string, password: string): Chainable<void>;
    }
  }
}
