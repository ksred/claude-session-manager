import { useState } from 'react'

function App() {
  const [count, setCount] = useState(0)

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="container mx-auto px-4 py-8">
        <header className="mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            Claude Session Manager
          </h1>
          <p className="text-lg text-gray-600">
            Manage your Claude API sessions efficiently
          </p>
        </header>

        <main>
          <div className="bg-white rounded-lg shadow-md p-6 max-w-md">
            <h2 className="text-2xl font-semibold text-gray-800 mb-4">
              Welcome
            </h2>
            <p className="text-gray-600 mb-4">
              This is your Claude Session Manager. Get started by configuring your sessions.
            </p>
            <div className="flex items-center space-x-4">
              <button
                onClick={() => setCount((count) => count + 1)}
                className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors duration-200 font-medium"
              >
                Count is {count}
              </button>
              <button
                onClick={() => setCount(0)}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors duration-200 font-medium"
              >
                Reset
              </button>
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}

export default App