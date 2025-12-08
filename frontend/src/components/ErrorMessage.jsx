export default function ErrorMessage({ message }) {
  if (!message) return null

  return (
    <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-center">
      {message}
    </div>
  )
}
