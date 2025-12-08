import Pagination from './Pagination'

export default function SearchResults({ results, query, currentPage, totalResults, onPageChange }) {
  if (results.length === 0 && query) {
    return (
      <div className="text-center text-gray-500 mt-12">
        <p className="text-lg">No results found for "{query}"</p>
        <p className="text-sm mt-2">Try a different search term</p>
      </div>
    )
  }

  if (results.length === 0) {
    return null
  }

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-semibold text-gray-800 mb-6">
        Search Results
      </h2>
      {results.map((result, index) => {
        // Clean up title - if it's just metadata/numbers, use URL as fallback
        const cleanTitle = result.Title && result.Title.trim() && result.Title.length > 3
          ? result.Title
          : result.URL

        // Clean up description/content
        const description = result.Content || result.Description
        const cleanDescription = description && description.trim() && description.length > 10
          ? description
          : null

        return (
          <div
            key={result.DocID || index}
            className="bg-white border border-light-blue-100 rounded-lg p-6 hover:shadow-lg hover:border-light-blue-300 transition-all"
          >
            <a
              href={result.URL}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xl font-semibold text-light-blue-500 hover:text-light-blue-600 hover:underline"
            >
              {cleanTitle}
            </a>
            <p className="text-sm text-green-600 mt-1 truncate">{result.URL}</p>
            {cleanDescription && (
              <p className="text-gray-700 mt-3 leading-relaxed line-clamp-3">
                {cleanDescription}
              </p>
            )}
          </div>
        )
      })}

      <Pagination
        currentPage={currentPage}
        totalResults={totalResults}
        onPageChange={onPageChange}
      />
    </div>
  )
}
