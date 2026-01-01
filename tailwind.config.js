/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.templ",
    "./templates/**/*_templ.go",
  ],
  theme: {
    extend: {
      fontFamily: {
        serif: ['Georgia', 'Times New Roman', 'Times', 'serif'],
      },
    },
  },
  plugins: [],
}

