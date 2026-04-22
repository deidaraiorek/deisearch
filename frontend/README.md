# Frontend

A React + Vite search interface that captures user queries, calls the query engine API, and renders paginated result pages.

## Architecture

**Flow:** User -> `SearchBar` -> `App` state -> `searchService.js` -> Query Engine API -> Results / Error / Pagination UI

**Components:**

- **App**: Owns loading, error, query, page, and result state
- **Header**: Displays the project branding
- **SearchBar**: Collects the search query and triggers requests
- **SearchResults**: Renders the result list returned by the API
- **Pagination**: Handles page changes and scroll reset
- **ErrorMessage**: Shows request failures
- **searchService**: Wraps the HTTP request to the backend API

## Usage

```bash
npm install
npm run dev
```

The frontend runs on the Vite dev server and sends requests to the query engine using `VITE_API_BASE_URL` or `http://localhost:8080` by default.

## How It Works

**Search Flow:**

- User submits a query from `SearchBar`
- `App` sets loading state and stores the current query/page
- `searchService.js` sends the request to the query engine
- Returned JSON updates result and pagination state
- UI rerenders either the landing state or the results state

**Current API Mode:**

- The current implementation calls `/semantic-search`
- Switching to classic keyword search only requires changing the request path in `src/services/searchService.js`

## Configuration

**Environment:**

- `VITE_API_BASE_URL`: Optional backend base URL override

**Default Backend Target:**

- `http://localhost:8080`
