# Frontend

React + TypeScript + Vite web app. The interface is Persian and right-to-left. It reads listings from the Go API at `/api/v1/listings` and marks fictional demo feeds clearly.

For standalone development, set `PORT=8081` for the API, then run `npm install` and `npm run dev` here. Vite proxies `/api` to the API. The Docker setup serves the built app through Nginx on port 5173.
