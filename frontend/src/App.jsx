import { useState } from 'react'
import Header from './components/Header'
import SearchBar from './components/SearchBar'
import SearchResults from './components/SearchResults'
import ErrorMessage from './components/ErrorMessage'
import { searchQuery } from './services/searchService'

function App() {
  const [results, setResults] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [currentQuery, setCurrentQuery] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [totalResults, setTotalResults] = useState(0)

  const handleSearch = async (query, page = 1) => {
    setLoading(true)
    setError(null)
    setCurrentQuery(query)
    setCurrentPage(page)

    try {
      const data = await searchQuery(query, page)
      setResults(data.results)
      setTotalResults(data.total)
    } catch (err) {
      setError('Failed to fetch results. Please try again.')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handlePageChange = (newPage) => {
    handleSearch(currentQuery, newPage)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const hasResults = results.length > 0 || error || currentQuery

  return (
    <div className="min-h-screen bg-white flex flex-col">
      {!hasResults && (
        <div className="flex-1 flex flex-col items-center justify-center px-4">
          <Header />
          <div className="w-full max-w-2xl mt-8">
            <SearchBar onSearch={handleSearch} loading={loading} />
          </div>
        </div>
      )}

      {hasResults && (
        <>
          <header className="py-4 border-b border-gray-100">
            <div className="container mx-auto px-4 flex items-center gap-8 max-w-6xl">
              <h1 className="text-xl font-semibold text-light-blue-500">DeiSearch</h1>
              <div className="flex-1 max-w-2xl">
                <SearchBar onSearch={handleSearch} loading={loading} />
              </div>
            </div>
          </header>

          <main className="container mx-auto px-4 py-6 max-w-4xl">
            <ErrorMessage message={error} />
            <SearchResults
              results={results}
              query={currentQuery}
              currentPage={currentPage}
              totalResults={totalResults}
              onPageChange={handlePageChange}
            />
          </main>
        </>
      )}
    </div>
  )
}

export default App
