import axios from 'axios'

function getConfig() {
    return {
        baseUrl: localStorage.getItem('nkudo_api_url') || '/api',
        adminKey: localStorage.getItem('nkudo_admin_key') || '',
        apiKey: localStorage.getItem('nkudo_api_key') || '',
        tenantId: localStorage.getItem('nkudo_tenant_id') || '',
    }
}

const http = axios.create({ timeout: 15000 })

http.interceptors.request.use((config) => {
    const cfg = getConfig()
    config.baseURL = cfg.baseUrl
    if (cfg.adminKey) config.headers['X-Admin-Key'] = cfg.adminKey
    if (cfg.apiKey) config.headers['X-API-Key'] = cfg.apiKey
    return config
})

http.interceptors.response.use(
    (res) => res,
    (error) => {
        if (axios.isAxiosError(error)) {
            const msg = error.response?.data?.error || error.message
            console.error(`[API] ${error.config?.method?.toUpperCase()} ${error.config?.url}: ${msg}`)
        }
        return Promise.reject(error)
    }
)

export default http
