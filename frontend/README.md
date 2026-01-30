This is a [Next.js](https://nextjs.org) project bootstrapped with [`create-next-app`](https://nextjs.org/docs/app/api-reference/cli/create-next-app).

## Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

### API and `.env`

- **Local dev (recommended):** Leave `NEXT_PUBLIC_API_URL` and `BACKEND_URL` commented in `.env`. The app will call `/api` (same origin), and Next.js will proxy those requests to the backend. No CORS, no 404 from wrong base URL.
- **If you set `NEXT_PUBLIC_API_URL=http://localhost:8080`:** The browser will call the backend directly (cross-origin). The backend must allow CORS; it is configured to allow `http://localhost:3000` and `http://127.0.0.1:3000` by default (set `CORS_ORIGINS` in the backend `.env` to add more). Use **no trailing slash** for the URL. Restart the Next dev server after changing `.env` so `NEXT_PUBLIC_*` is picked up.
- **If you still get 404:** Restart `pnpm dev` after editing `.env`, and confirm the backend is listening on the port you use (e.g. 8080).

You can start editing the page by modifying `app/page.tsx`. The page auto-updates as you edit the file.

This project uses [`next/font`](https://nextjs.org/docs/app/building-your-application/optimizing/fonts) to automatically optimize and load [Geist](https://vercel.com/font), a new font family for Vercel.

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js) - your feedback and contributions are welcome!

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/app/building-your-application/deploying) for more details.
