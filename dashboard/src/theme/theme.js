import { createTheme } from '@mui/material/styles';

// Dark theme optimized for SOC environments
const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#667eea',
      light: '#a5b4fc',
      dark: '#4c51bf',
      contrastText: '#ffffff',
    },
    secondary: {
      main: '#764ba2',
      light: '#a78bfa',
      dark: '#5b21b6',
    },
    background: {
      default: '#0f172a',      // Very dark blue-gray
      paper: '#1e293b',        // Dark blue-gray for cards
    },
    text: {
      primary: '#f1f5f9',      // Almost white
      secondary: '#cbd5e1',    // Light gray
      disabled: '#64748b',     // Medium gray
    },
    error: {
      main: '#ef4444',
      light: '#fca5a5',
      dark: '#b91c1c',
    },
    warning: {
      main: '#f59e0b',
      light: '#fcd34d',
      dark: '#d97706',
    },
    info: {
      main: '#3b82f6',
      light: '#93c5fd',
      dark: '#1e40af',
    },
    success: {
      main: '#10b981',
      light: '#6ee7b7',
      dark: '#047857',
    },
    // Custom severity colors
    severity: {
      critical: '#ef4444',
      high: '#f59e0b',
      medium: '#fbbf24',
      low: '#10b981',
    },
  },
  typography: {
    fontFamily: [
      'Inter',
      '-apple-system',
      'BlinkMacSystemFont',
      'Segoe UI',
      'Roboto',
      'Helvetica Neue',
      'Arial',
      'sans-serif',
    ].join(','),
    h1: {
      fontSize: '2.5rem',
      fontWeight: 700,
      lineHeight: 1.2,
    },
    h2: {
      fontSize: '2rem',
      fontWeight: 600,
      lineHeight: 1.3,
    },
    h3: {
      fontSize: '1.75rem',
      fontWeight: 600,
      lineHeight: 1.4,
    },
    h4: {
      fontSize: '1.5rem',
      fontWeight: 600,
      lineHeight: 1.4,
    },
    h5: {
      fontSize: '1.25rem',
      fontWeight: 600,
      lineHeight: 1.5,
    },
    h6: {
      fontSize: '1rem',
      fontWeight: 600,
      lineHeight: 1.6,
    },
    subtitle1: {
      fontSize: '1rem',
      lineHeight: 1.75,
    },
    subtitle2: {
      fontSize: '0.875rem',
      lineHeight: 1.57,
    },
    body1: {
      fontSize: '1rem',
      lineHeight: 1.5,
    },
    body2: {
      fontSize: '0.875rem',
      lineHeight: 1.43,
    },
    button: {
      textTransform: 'none',
      fontWeight: 600,
    },
    caption: {
      fontSize: '0.75rem',
      lineHeight: 1.66,
    },
    overline: {
      fontSize: '0.75rem',
      fontWeight: 600,
      lineHeight: 2.66,
      textTransform: 'uppercase',
    },
  },
  shape: {
    borderRadius: 12,
  },
  shadows: [
    'none',
    '0px 2px 4px rgba(0,0,0,0.2)',
    '0px 4px 8px rgba(0,0,0,0.2)',
    '0px 8px 16px rgba(0,0,0,0.2)',
    '0px 12px 24px rgba(0,0,0,0.2)',
    '0px 16px 32px rgba(0,0,0,0.2)',
    '0px 20px 40px rgba(0,0,0,0.2)',
    '0px 24px 48px rgba(0,0,0,0.2)',
    '0px 1px 3px rgba(0,0,0,0.12)',
    '0px 1px 5px rgba(0,0,0,0.2)',
    '0px 1px 8px rgba(0,0,0,0.24)',
    '0px 1px 10px rgba(0,0,0,0.28)',
    '0px 1px 14px rgba(0,0,0,0.32)',
    '0px 1px 18px rgba(0,0,0,0.36)',
    '0px 6px 10px rgba(0,0,0,0.14)',
    '0px 7px 10px rgba(0,0,0,0.16)',
    '0px 8px 10px rgba(0,0,0,0.18)',
    '0px 9px 12px rgba(0,0,0,0.2)',
    '0px 10px 14px rgba(0,0,0,0.22)',
    '0px 11px 15px rgba(0,0,0,0.24)',
    '0px 12px 17px rgba(0,0,0,0.26)',
    '0px 13px 19px rgba(0,0,0,0.28)',
    '0px 14px 21px rgba(0,0,0,0.3)',
    '0px 15px 22px rgba(0,0,0,0.32)',
    '0px 16px 24px rgba(0,0,0,0.34)',
  ],
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          padding: '10px 20px',
          fontSize: '0.875rem',
        },
        contained: {
          boxShadow: 'none',
          '&:hover': {
            boxShadow: '0px 4px 8px rgba(0,0,0,0.2)',
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 12,
          boxShadow: '0px 4px 8px rgba(0,0,0,0.2)',
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        },
        elevation1: {
          boxShadow: '0px 2px 4px rgba(0,0,0,0.2)',
        },
        elevation2: {
          boxShadow: '0px 4px 8px rgba(0,0,0,0.2)',
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          boxShadow: 'none',
          borderBottom: '1px solid rgba(255, 255, 255, 0.1)',
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          borderRight: '1px solid rgba(255, 255, 255, 0.1)',
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottom: '1px solid rgba(255, 255, 255, 0.05)',
        },
        head: {
          fontWeight: 600,
          backgroundColor: 'rgba(255, 255, 255, 0.02)',
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 6,
        },
      },
    },
  },
});

export default theme;
