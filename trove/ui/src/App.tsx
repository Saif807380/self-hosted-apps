import { Routes, Route, Navigate } from 'react-router-dom'
import { Box } from '@chakra-ui/react'
import Layout from './components/Layout'
import BooksPage from './pages/BooksPage'
import GamesPage from './pages/GamesPage'
import TravelPage from './pages/TravelPage'
import WorkoutsPage from './pages/WorkoutsPage'

export default function App() {
  return (
    <Box minH="100vh" bg="bg.canvas">
      <Layout>
        <Routes>
          <Route path="/" element={<Navigate to="/books" replace />} />
          <Route path="/books" element={<BooksPage />} />
          <Route path="/games" element={<GamesPage />} />
          <Route path="/travel" element={<TravelPage />} />
          <Route path="/workouts" element={<WorkoutsPage />} />
        </Routes>
      </Layout>
    </Box>
  )
}
