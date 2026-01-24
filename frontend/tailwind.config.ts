import type { Config } from 'tailwindcss';
import { fontFamily } from 'tailwindcss/defaultTheme';

const config: Config = {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	darkMode: 'class',
	theme: {
		extend: {
			screens: {
				xs: '450px'
			},
			fontFamily: {
				sans: ['Inter', ...fontFamily.sans],
				mono: ['"JetBrains Mono"', ...fontFamily.mono]
			},
			colors: {
				primary: 'rgb(var(--color-primary-rgb) / <alpha-value>)',
				secondary: 'rgb(var(--color-secondary-rgb) / <alpha-value>)',
				tertiary: 'rgb(var(--color-tertiary-rgb) / <alpha-value>)',
				accent: 'rgb(var(--color-accent-rgb) / <alpha-value>)',
				'accent-hover': 'rgb(var(--color-accent-hover-rgb) / <alpha-value>)',
				text: {
					primary: 'rgb(var(--color-text-primary-rgb) / <alpha-value>)',
					secondary: 'rgb(var(--color-text-secondary-rgb) / <alpha-value>)'
				}
			},
			keyframes: {
				'fade-in-up': {
					'0%': { opacity: '0', transform: 'translateY(10px) scale(0.98)' },
					'100%': { opacity: '1', transform: 'translateY(0) scale(1)' }
				},
				shimmer: {
					'100%': { transform: 'translateX(100%)' }
				},
				'gradient-spin': {
					'0%': { backgroundPosition: '0% 50%' },
					'50%': { backgroundPosition: '100% 50%' },
					'100%': { backgroundPosition: '0% 50%' }
				},
				'gradient-x': {
					'0%, 100%': {
						'background-size': '200% 200%',
						'background-position': 'left center'
					},
					'50%': {
						'background-size': '200% 200%',
						'background-position': 'right center'
					}
				},
				shine: {
					'0%': { backgroundPosition: '200% 0' },
					'100%': { backgroundPosition: '-200% 0' }
				},
				float: {
					'0%, 100%': { transform: 'translateY(0)' },
					'50%': { transform: 'translateY(-10px)' }
				}
			},
			animation: {
				'fade-in-up': 'fade-in-up 0.4s ease-out forwards',
				shimmer: 'shimmer 1.5s infinite',
				'gradient-spin': 'gradient-spin 4s linear infinite',
				'gradient-x': 'gradient-x 3s ease infinite',
				shine: 'shine 3s linear infinite',
				'pulse-slow': 'pulse 4s cubic-bezier(0.4, 0, 0.6, 1) infinite',
				'spin-slow': 'spin 12s linear infinite',
				float: 'float 5s ease-in-out infinite',
				scanline: 'scanline 8s linear infinite'
			},
			keyframes: {
				'fade-in-up': {
					'0%': { opacity: '0', transform: 'translateY(10px) scale(0.98)' },
					'100%': { opacity: '1', transform: 'translateY(0) scale(1)' }
				},
				shimmer: {
					'100%': { transform: 'translateX(100%)' }
				},
				'gradient-spin': {
					'0%': { backgroundPosition: '0% 50%' },
					'50%': { backgroundPosition: '100% 50%' },
					'100%': { backgroundPosition: '0% 50%' }
				},
				'gradient-x': {
					'0%, 100%': {
						'background-size': '200% 200%',
						'background-position': 'left center'
					},
					'50%': {
						'background-size': '200% 200%',
						'background-position': 'right center'
					}
				},
				shine: {
					'0%': { backgroundPosition: '200% 0' },
					'100%': { backgroundPosition: '-200% 0' }
				},
				float: {
					'0%, 100%': { transform: 'translateY(0)' },
					'50%': { transform: 'translateY(-10px)' }
				},
				scanline: {
					'0%': { backgroundPosition: '0% 0%' },
					'100%': { backgroundPosition: '0% 100%' }
				}
			}
		}
	},
	plugins: [
		require('@tailwindcss/typography'),
		function ({ addUtilities }: { addUtilities: any }) {
			addUtilities({
				'.perspective-1000': {
					perspective: '1000px',
				},
				'.preserve-3d': {
					transformStyle: 'preserve-3d',
				},
				'.scrollbar-hide': {
					'-ms-overflow-style': 'none',
					'scrollbar-width': 'none',
					'&::-webkit-scrollbar': {
						display: 'none',
					},
				},
			});
		},
	]
};

export default config;
