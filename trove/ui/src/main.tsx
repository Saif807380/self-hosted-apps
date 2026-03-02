import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ChakraProvider } from '@chakra-ui/react'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import { system } from './theme/theme'
import { AppToaster } from './components/AppToaster'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <ChakraProvider value={system}>
          <App />
          <AppToaster />
      </ChakraProvider>
    </BrowserRouter>
  </StrictMode>,
)
