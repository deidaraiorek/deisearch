const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

export async function searchQuery(query, page = 1) {
  const response = await fetch(`${API_BASE_URL}/search?q=${encodeURIComponent(query)}&page=${page}`)

  if (!response.ok) {
    throw new Error('Search failed')
  }

  const data = await response.json()
  return {
    results: data.results || [],
    total: data.total || 0,
    page: data.page || 1,
    query: data.query || query
  }
}
