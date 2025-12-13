#!/usr/bin/env node

/**
 * 01-seed-users.mjs
 * Seeds tenant and initial users for local development
 *
 * Usage: node scripts/01-seed-users.mjs
 *
 * Creates:
 * - Tenant: Booking Rush (default tenant)
 * - super_admin: admin@admin.com / Admin123!
 * - organizer: organizer@test.com / Test123!
 * - customer: customer@test.com / Test123!
 */

import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

// Configuration
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080/api/v1';
const DB_CONTAINER = process.env.DB_CONTAINER || 'booking-rush-postgres';
const DB_NAME = process.env.AUTH_DATABASE_DBNAME || 'auth_db';
const DB_USER = process.env.AUTH_DATABASE_USER || 'postgres';

// Tenant to seed
const TENANT = {
  id: '00000000-0000-0000-0000-000000000001',
  name: 'Booking Rush',
  slug: 'booking-rush'
};

// Users to seed
const USERS = [
  {
    email: 'admin@admin.com',
    password: 'Admin123!',
    name: 'Super Admin',
    role: 'super_admin'
  },
  {
    email: 'organizer@test.com',
    password: 'Test123!',
    name: 'Test Organizer',
    role: 'organizer'
  },
  {
    email: 'customer@test.com',
    password: 'Test123!',
    name: 'Test Customer',
    role: 'customer'
  }
];

// Colors for console
const colors = {
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
  reset: '\x1b[0m'
};

function log(color, message) {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

async function runSQL(sql) {
  try {
    const cmd = `docker exec ${DB_CONTAINER} psql -U ${DB_USER} -d ${DB_NAME} -c "${sql.replace(/"/g, '\\"')}"`;
    const { stdout } = await execAsync(cmd);
    return { success: true, output: stdout };
  } catch (error) {
    return { success: false, error: error.message };
  }
}

async function checkApiHealth() {
  log('yellow', '\n[Step 1] Checking API health...');

  try {
    const res = await fetch(`${API_BASE_URL.replace('/api/v1', '')}/health`);
    const data = await res.json();

    if (data.status === 'healthy') {
      log('green', '  API Gateway is healthy');
      return true;
    }
  } catch (error) {
    log('red', `  API not available: ${error.message}`);
    return false;
  }
  return false;
}

async function createTenant() {
  log('yellow', '\n[Step 2] Creating tenant...');

  // Check if tenant exists
  const checkResult = await runSQL(`SELECT id FROM tenants WHERE slug = '${TENANT.slug}'`);

  if (checkResult.output && checkResult.output.includes(TENANT.id)) {
    log('yellow', `  Tenant already exists: ${TENANT.name}`);
    return true;
  }

  // Create tenant
  const sql = `INSERT INTO tenants (id, name, slug, is_active) VALUES ('${TENANT.id}', '${TENANT.name}', '${TENANT.slug}', true) ON CONFLICT (slug) DO NOTHING`;
  const result = await runSQL(sql);

  if (result.success) {
    log('green', `  Created tenant: ${TENANT.name} (${TENANT.slug})`);
    return true;
  } else {
    log('red', `  Failed to create tenant: ${result.error}`);
    return false;
  }
}

async function registerUser(user) {
  try {
    const res = await fetch(`${API_BASE_URL}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: user.email,
        password: user.password,
        name: user.name
      })
    });

    const data = await res.json();

    if (data.success) {
      log('green', `  Registered: ${user.email}`);
      return { success: true, userId: data.data.user.id };
    } else if (data.error?.code === 'USER_ALREADY_EXISTS') {
      log('yellow', `  Already exists: ${user.email}`);
      return { success: true, exists: true };
    } else {
      log('red', `  Failed to register ${user.email}: ${JSON.stringify(data.error)}`);
      return { success: false };
    }
  } catch (error) {
    log('red', `  Error registering ${user.email}: ${error.message}`);
    return { success: false };
  }
}

async function updateUserRoleAndTenant(email, role) {
  // Update role and tenant_id
  const sql = `UPDATE users SET role = '${role}', tenant_id = '${TENANT.id}' WHERE email = '${email}'`;
  const result = await runSQL(sql);

  if (result.success && result.output.includes('UPDATE 1')) {
    log('green', `  Updated: ${email} -> role=${role}, tenant=${TENANT.slug}`);
    return true;
  } else if (result.output && result.output.includes('UPDATE 0')) {
    log('yellow', `  User not found: ${email}`);
    return false;
  } else {
    log('red', `  Failed to update ${email}`);
    return false;
  }
}

async function verifyLogin(user) {
  try {
    const res = await fetch(`${API_BASE_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: user.email,
        password: user.password
      })
    });

    const data = await res.json();

    if (data.success) {
      const tenantInfo = data.data.user.tenant_id ? `tenant=${data.data.user.tenant_id.substring(0, 8)}...` : 'no tenant';
      log('green', `  Login OK: ${user.email} (role=${data.data.user.role}, ${tenantInfo})`);
      return true;
    } else {
      log('red', `  Login failed: ${user.email}`);
      return false;
    }
  } catch (error) {
    log('red', `  Error verifying login: ${error.message}`);
    return false;
  }
}

async function main() {
  console.log('');
  log('cyan', '='.repeat(60));
  log('cyan', '  Booking Rush - Seed Users Script');
  log('cyan', '='.repeat(60));
  console.log('');
  log('blue', `API URL: ${API_BASE_URL}`);
  log('blue', `DB Container: ${DB_CONTAINER}`);
  log('blue', `Tenant: ${TENANT.name} (${TENANT.slug})`);
  console.log('');

  // Step 1: Check API health
  const isHealthy = await checkApiHealth();
  if (!isHealthy) {
    log('red', '\nAPI is not available. Please start the services first:');
    log('yellow', '  docker-compose -f docker-compose.db.yml up -d');
    log('yellow', '  docker-compose -f docker-compose.services.yml up -d');
    process.exit(1);
  }

  // Step 2: Create tenant
  const tenantCreated = await createTenant();
  if (!tenantCreated) {
    log('red', '\nFailed to create tenant. Exiting.');
    process.exit(1);
  }

  // Step 3: Register users
  log('yellow', '\n[Step 3] Registering users...');
  for (const user of USERS) {
    await registerUser(user);
  }

  // Step 4: Update roles and tenant
  log('yellow', '\n[Step 4] Updating user roles and tenant...');
  for (const user of USERS) {
    await updateUserRoleAndTenant(user.email, user.role);
  }

  // Step 5: Verify logins
  log('yellow', '\n[Step 5] Verifying logins...');
  for (const user of USERS) {
    await verifyLogin(user);
  }

  // Summary
  console.log('');
  log('cyan', '='.repeat(60));
  log('green', '  Seed completed!');
  log('cyan', '='.repeat(60));
  console.log('');
  log('blue', 'Tenant:');
  console.log(`  ${TENANT.name} (${TENANT.slug})`);
  console.log('');
  log('blue', 'Test accounts:');
  console.log('');
  console.log('  | Role        | Email              | Password  |');
  console.log('  |-------------|--------------------|-----------| ');
  for (const user of USERS) {
    console.log(`  | ${user.role.padEnd(11)} | ${user.email.padEnd(18)} | ${user.password.padEnd(9)} |`);
  }
  console.log('');
  log('blue', 'Next step:');
  console.log('  ADMIN_EMAIL="organizer@test.com" ADMIN_PASSWORD="Test123!" node scripts/02-seed-events.mjs');
  console.log('');
}

main().catch(console.error);
