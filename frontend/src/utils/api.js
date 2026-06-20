import axios from 'axios'

const USERNAME_KEY = 'bim_viewer_username'

export function getCurrentUsername() {
  return localStorage.getItem(USERNAME_KEY) || ''
}

export function setCurrentUsername(username) {
  localStorage.setItem(USERNAME_KEY, username)
}

export function clearCurrentUsername() {
  localStorage.removeItem(USERNAME_KEY)
}

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 300000,
  headers: {
    'Content-Type': 'application/json'
  }
})

api.interceptors.request.use(
  config => {
    const username = getCurrentUsername()
    if (username) {
      config.headers['X-Current-User'] = username
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

api.interceptors.response.use(
  response => response,
  error => {
    console.error('API Error:', error.response?.data || error.message)
    return Promise.reject(error)
  }
)

export default api
