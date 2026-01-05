import { ApiClient, randomEmail } from './client'

describe('Auth Flow', () => {
  const api = new ApiClient()
  const testEmail = randomEmail()
  const testPassword = process.env.TEST_PASSWORD || 'Test123!'
  const testName = process.env.TEST_NAME || 'Test User'
  let accessToken: string

  describe('POST /auth/register', () => {
    it('should register a new user successfully', async () => {
      const response = await api.register(testEmail, testPassword, testName)

      expect(response.success).toBe(true)
      expect(response.data).toHaveProperty('access_token')
      expect(response.data).toHaveProperty('refresh_token')
      expect(response.data).toHaveProperty('user')
      expect(response.data.user.email).toBe(testEmail)
      expect(response.data.user.name).toBe(testName)

      accessToken = response.data.access_token
    })

    it('should reject duplicate email registration', async () => {
      try {
        await api.register(testEmail, testPassword, testName)
        fail('Should have thrown an error')
      } catch (error: any) {
        expect(error.response.status).toBe(400)
      }
    })

    it('should reject weak password', async () => {
      try {
        await api.register(randomEmail(), '123', testName)
        fail('Should have thrown an error')
      } catch (error: any) {
        expect(error.response.status).toBeGreaterThanOrEqual(400)
      }
    })

    it('should reject invalid email format', async () => {
      try {
        await api.register('invalid-email', testPassword, testName)
        fail('Should have thrown an error')
      } catch (error: any) {
        expect(error.response.status).toBeGreaterThanOrEqual(400)
      }
    })
  })

  describe('POST /auth/login', () => {
    it('should login with valid credentials', async () => {
      const response = await api.login(testEmail, testPassword)

      expect(response.success).toBe(true)
      expect(response.data).toHaveProperty('access_token')
      expect(response.data).toHaveProperty('refresh_token')
      expect(response.data.user.email).toBe(testEmail)

      accessToken = response.data.access_token
    })

    it('should reject invalid password', async () => {
      try {
        await api.login(testEmail, 'wrongpassword')
        fail('Should have thrown an error')
      } catch (error: any) {
        expect(error.response.status).toBe(401)
      }
    })

    it('should reject non-existent user', async () => {
      try {
        await api.login('nonexistent@test.com', testPassword)
        fail('Should have thrown an error')
      } catch (error: any) {
        expect(error.response.status).toBe(401)
      }
    })
  })

  describe('GET /auth/me', () => {
    it('should get current user with valid token', async () => {
      api.setToken(accessToken)
      const response = await api.getCurrentUser()

      expect(response.success).toBe(true)
      expect(response.data.email).toBe(testEmail)
      expect(response.data.name).toBe(testName)
    })

    it('should reject request without token', async () => {
      const unauthApi = new ApiClient()
      try {
        await unauthApi.getCurrentUser()
        fail('Should have thrown an error')
      } catch (error: any) {
        expect(error.response.status).toBe(401)
      }
    })

    it('should reject request with invalid token', async () => {
      const invalidApi = new ApiClient('invalid-token')
      try {
        await invalidApi.getCurrentUser()
        fail('Should have thrown an error')
      } catch (error: any) {
        expect(error.response.status).toBe(401)
      }
    })
  })
})
