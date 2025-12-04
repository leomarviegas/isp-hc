/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./index.html", "./src/**/*.{js,jsx,ts,tsx}"],
  theme: {
    extend: {
      colors: {
        accent: "#0ea5e9",
        ink: "#0f172a"
      }
    }
  },
  plugins: [],
};
