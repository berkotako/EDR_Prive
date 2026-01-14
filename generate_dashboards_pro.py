#!/usr/bin/env python3
"""
Professional Dashboard Mockup Generator for Privé EDR/DLP Platform
Generates high-quality, attractive dashboard images for marketing
"""

import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
from matplotlib.patches import FancyBboxPatch, Circle, Rectangle
import numpy as np
from datetime import datetime, timedelta
import warnings
warnings.filterwarnings('ignore')

# Professional color palette - Dark theme with cyber security aesthetics
COLORS = {
    'bg_dark': '#0a0e1a',
    'bg_card': '#141b2d',
    'bg_card_hover': '#1a2332',
    'primary': '#6366f1',  # Indigo
    'secondary': '#8b5cf6',  # Purple
    'accent': '#10b981',  # Emerald
    'warning': '#f59e0b',  # Amber
    'danger': '#ef4444',  # Red
    'text_primary': '#e5e7eb',
    'text_secondary': '#9ca3af',
    'border': '#1f2937',
    'grid': '#1f2937',
    'critical': '#dc2626',
    'high': '#f97316',
    'medium': '#facc15',
    'low': '#22c55e',
}

# Set global style
plt.style.use('dark_background')
plt.rcParams['font.family'] = 'sans-serif'
plt.rcParams['font.sans-serif'] = ['Arial', 'Helvetica', 'DejaVu Sans']
plt.rcParams['font.size'] = 9
plt.rcParams['axes.facecolor'] = COLORS['bg_card']
plt.rcParams['figure.facecolor'] = COLORS['bg_dark']
plt.rcParams['axes.edgecolor'] = COLORS['border']
plt.rcParams['grid.color'] = COLORS['grid']
plt.rcParams['text.color'] = COLORS['text_primary']
plt.rcParams['axes.labelcolor'] = COLORS['text_primary']
plt.rcParams['xtick.color'] = COLORS['text_secondary']
plt.rcParams['ytick.color'] = COLORS['text_secondary']

def add_card(ax, title, value=None, subtitle=None, trend=None):
    """Add a professional metric card"""
    ax.set_xlim(0, 1)
    ax.set_ylim(0, 1)
    ax.axis('off')

    # Card background
    card = FancyBboxPatch((0.05, 0.1), 0.9, 0.8,
                           boxstyle="round,pad=0.05",
                           facecolor=COLORS['bg_card'],
                           edgecolor=COLORS['border'],
                           linewidth=2,
                           alpha=0.9)
    ax.add_patch(card)

    # Title
    ax.text(0.5, 0.75, title, ha='center', va='center',
            fontsize=10, color=COLORS['text_secondary'], weight='normal')

    # Value
    if value:
        ax.text(0.5, 0.45, value, ha='center', va='center',
                fontsize=28, color=COLORS['text_primary'], weight='bold')

    # Subtitle
    if subtitle:
        color = COLORS['accent'] if trend and trend > 0 else COLORS['text_secondary']
        ax.text(0.5, 0.25, subtitle, ha='center', va='center',
                fontsize=9, color=color, weight='normal')

def generate_soc_dashboard():
    """Generate SOC Operations Center Dashboard"""
    fig = plt.figure(figsize=(16, 10), facecolor=COLORS['bg_dark'])

    # Title
    fig.suptitle('Security Operations Center - Live Threat Monitoring',
                 fontsize=20, fontweight='bold', color=COLORS['text_primary'], y=0.98)
    fig.text(0.5, 0.95, 'Real-time endpoint protection across 10,247 agents',
             ha='center', fontsize=11, color=COLORS['text_secondary'])

    # Create grid
    gs = fig.add_gridspec(4, 6, hspace=0.4, wspace=0.3, top=0.92, bottom=0.05, left=0.05, right=0.95)

    # Row 1: Key Metrics Cards
    ax1 = fig.add_subplot(gs[0, :2])
    add_card(ax1, 'ACTIVE THREATS', '23', '↓ 15% vs yesterday', trend=-15)

    ax2 = fig.add_subplot(gs[0, 2:4])
    add_card(ax2, 'EVENTS/SECOND', '12.5K', '↑ 3.2% vs avg', trend=3.2)

    ax3 = fig.add_subplot(gs[0, 4:])
    add_card(ax3, 'AGENTS ONLINE', '10,247', '99.8% uptime', trend=0)

    # Row 2: Threat Timeline
    ax4 = fig.add_subplot(gs[1, :])
    ax4.set_facecolor(COLORS['bg_card'])
    ax4.set_title('Threat Detection Timeline (Last 24 Hours)', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    hours = np.arange(24)
    critical = np.random.poisson(2, 24)
    high = np.random.poisson(5, 24)
    medium = np.random.poisson(12, 24)
    low = np.random.poisson(20, 24)

    ax4.fill_between(hours, 0, critical, alpha=0.9, color=COLORS['critical'], label='Critical')
    ax4.fill_between(hours, critical, critical+high, alpha=0.8, color=COLORS['high'], label='High')
    ax4.fill_between(hours, critical+high, critical+high+medium, alpha=0.7, color=COLORS['medium'], label='Medium')
    ax4.fill_between(hours, critical+high+medium, critical+high+medium+low, alpha=0.6, color=COLORS['low'], label='Low')

    ax4.set_xlabel('Hour of Day', fontsize=10, color=COLORS['text_secondary'])
    ax4.set_ylabel('Threat Count', fontsize=10, color=COLORS['text_secondary'])
    ax4.legend(loc='upper left', framealpha=0.9, facecolor=COLORS['bg_card_hover'])
    ax4.grid(True, alpha=0.2, linestyle='--')
    ax4.spines['top'].set_visible(False)
    ax4.spines['right'].set_visible(False)

    # Row 3: MITRE ATT&CK Heat Map
    ax5 = fig.add_subplot(gs[2, :4])
    ax5.set_facecolor(COLORS['bg_card'])
    ax5.set_title('MITRE ATT&CK Tactics Coverage', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    tactics = ['Initial\nAccess', 'Execution', 'Persistence', 'Priv Esc', 'Defense\nEvasion',
               'Credential\nAccess', 'Discovery', 'Lateral\nMovement', 'Collection', 'Exfiltration']
    detections = [8, 15, 12, 6, 22, 4, 18, 3, 7, 2]
    colors_bar = [COLORS['danger'] if d > 15 else COLORS['warning'] if d > 8 else COLORS['accent'] for d in detections]

    bars = ax5.barh(tactics, detections, color=colors_bar, alpha=0.8, edgecolor=COLORS['border'], linewidth=1.5)

    # Add value labels
    for i, (bar, val) in enumerate(zip(bars, detections)):
        ax5.text(val + 0.5, i, f'{val}', va='center', fontsize=10, color=COLORS['text_primary'], weight='bold')

    ax5.set_xlabel('Detections Today', fontsize=10, color=COLORS['text_secondary'])
    ax5.grid(axis='x', alpha=0.2, linestyle='--')
    ax5.spines['top'].set_visible(False)
    ax5.spines['right'].set_visible(False)
    ax5.set_xlim(0, max(detections) + 5)

    # Row 3: Severity Distribution (Donut Chart)
    ax6 = fig.add_subplot(gs[2, 4:])
    ax6.set_facecolor(COLORS['bg_card'])
    ax6.set_title('Alert Severity Distribution', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    severities = [23, 45, 87, 145]
    severity_labels = ['Critical', 'High', 'Medium', 'Low']
    severity_colors = [COLORS['critical'], COLORS['high'], COLORS['medium'], COLORS['low']]

    wedges, texts, autotexts = ax6.pie(severities, labels=severity_labels, autopct='%1.1f%%',
                                        colors=severity_colors, startangle=90,
                                        wedgeprops=dict(width=0.4, edgecolor=COLORS['border'], linewidth=2),
                                        textprops=dict(color=COLORS['text_primary'], fontsize=10, weight='bold'))

    # Row 4: Top Affected Hosts
    ax7 = fig.add_subplot(gs[3, :3])
    ax7.set_facecolor(COLORS['bg_card'])
    ax7.set_title('Top 5 Affected Endpoints', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    hosts = ['DESKTOP-A4F21', 'LAPTOP-8B92E', 'SERVER-DC01', 'WORKSTATION-45', 'DEVBOX-STAGING']
    incidents = [18, 14, 12, 9, 7]

    bars2 = ax7.barh(hosts, incidents, color=COLORS['danger'], alpha=0.8, edgecolor=COLORS['border'], linewidth=1.5)

    for i, (bar, val) in enumerate(zip(bars2, incidents)):
        ax7.text(val + 0.3, i, f'{val} alerts', va='center', fontsize=9, color=COLORS['text_primary'])

    ax7.set_xlabel('Alert Count', fontsize=10, color=COLORS['text_secondary'])
    ax7.grid(axis='x', alpha=0.2, linestyle='--')
    ax7.spines['top'].set_visible(False)
    ax7.spines['right'].set_visible(False)
    ax7.set_xlim(0, max(incidents) + 5)

    # Row 4: Recent Critical Alerts
    ax8 = fig.add_subplot(gs[3, 3:])
    ax8.set_facecolor(COLORS['bg_card'])
    ax8.set_title('Recent Critical Alerts', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)
    ax8.axis('off')

    alerts = [
        ('15:42', 'Ransomware Activity Detected', 'LAPTOP-8B92E'),
        ('14:18', 'Lateral Movement Attempt', 'SERVER-DC01'),
        ('12:33', 'Privilege Escalation', 'DESKTOP-A4F21'),
        ('11:05', 'C2 Beacon Communication', 'WORKSTATION-45'),
    ]

    y_pos = 0.85
    for time, alert, host in alerts:
        ax8.text(0.05, y_pos, f'🔴 {time}', fontsize=9, color=COLORS['critical'], weight='bold')
        ax8.text(0.05, y_pos - 0.08, alert, fontsize=9, color=COLORS['text_primary'])
        ax8.text(0.05, y_pos - 0.14, f'Host: {host}', fontsize=8, color=COLORS['text_secondary'], style='italic')
        y_pos -= 0.23

    plt.savefig('docs/images/dashboard-soc.png', dpi=150, bbox_inches='tight', facecolor=COLORS['bg_dark'])
    print("✅ Generated: dashboard-soc.png")
    plt.close()

def generate_threat_hunting_dashboard():
    """Generate Threat Hunting Workbench Dashboard"""
    fig = plt.figure(figsize=(16, 10), facecolor=COLORS['bg_dark'])

    fig.suptitle('Threat Hunting Workbench - Advanced Investigation',
                 fontsize=20, fontweight='bold', color=COLORS['text_primary'], y=0.98)
    fig.text(0.5, 0.95, 'Query billions of events in <100ms with ClickHouse analytics',
             ha='center', fontsize=11, color=COLORS['text_secondary'])

    gs = fig.add_gridspec(4, 4, hspace=0.4, wspace=0.3, top=0.92, bottom=0.05, left=0.05, right=0.95)

    # Query Stats Cards
    ax1 = fig.add_subplot(gs[0, 0])
    add_card(ax1, 'QUERY TIME', '47ms', 'Lightning fast')

    ax2 = fig.add_subplot(gs[0, 1])
    add_card(ax2, 'RESULTS', '1,247', 'events found')

    ax3 = fig.add_subplot(gs[0, 2])
    add_card(ax3, 'TIME RANGE', '24h', 'scanning period')

    ax4 = fig.add_subplot(gs[0, 3])
    add_card(ax4, 'DATA SCANNED', '2.4TB', 'compressed')

    # Event Timeline
    ax5 = fig.add_subplot(gs[1, :])
    ax5.set_facecolor(COLORS['bg_card'])
    ax5.set_title('Event Timeline - Process Creation & Network Connections', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    times = np.linspace(0, 24, 200)
    process_events = np.sin(times * 0.5) * 20 + 40 + np.random.normal(0, 5, 200)
    network_events = np.cos(times * 0.7) * 15 + 35 + np.random.normal(0, 3, 200)

    ax5.plot(times, process_events, color=COLORS['primary'], linewidth=2, label='Process Creation', alpha=0.9)
    ax5.plot(times, network_events, color=COLORS['accent'], linewidth=2, label='Network Connections', alpha=0.9)
    ax5.fill_between(times, process_events, alpha=0.2, color=COLORS['primary'])
    ax5.fill_between(times, network_events, alpha=0.2, color=COLORS['accent'])

    ax5.set_xlabel('Hour of Day', fontsize=10, color=COLORS['text_secondary'])
    ax5.set_ylabel('Events per Minute', fontsize=10, color=COLORS['text_secondary'])
    ax5.legend(loc='upper right', framealpha=0.9, facecolor=COLORS['bg_card_hover'])
    ax5.grid(True, alpha=0.2, linestyle='--')
    ax5.spines['top'].set_visible(False)
    ax5.spines['right'].set_visible(False)

    # Process Tree Visualization
    ax6 = fig.add_subplot(gs[2:, :2])
    ax6.set_facecolor(COLORS['bg_card'])
    ax6.set_title('Process Execution Tree', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)
    ax6.set_xlim(0, 10)
    ax6.set_ylim(0, 10)
    ax6.axis('off')

    # Draw process tree
    processes = [
        (5, 9, 'explorer.exe', COLORS['accent']),
        (3, 7, 'cmd.exe', COLORS['warning']),
        (7, 7, 'powershell.exe', COLORS['danger']),
        (2, 5, 'certutil.exe', COLORS['critical']),
        (4, 5, 'net.exe', COLORS['danger']),
        (6, 5, 'reg.exe', COLORS['danger']),
        (8, 5, 'rundll32.exe', COLORS['critical']),
    ]

    # Draw connections
    connections = [(5, 9, 3, 7), (5, 9, 7, 7), (3, 7, 2, 5), (3, 7, 4, 5), (7, 7, 6, 5), (7, 7, 8, 5)]
    for x1, y1, x2, y2 in connections:
        ax6.plot([x1, x2], [y1-0.3, y2+0.3], color=COLORS['text_secondary'], linewidth=1.5, alpha=0.5, linestyle='--')

    # Draw process nodes
    for x, y, name, color in processes:
        circle = Circle((x, y), 0.4, facecolor=color, edgecolor=COLORS['border'], linewidth=2, alpha=0.8)
        ax6.add_patch(circle)
        ax6.text(x, y, name.split('.')[0][:8], ha='center', va='center', fontsize=8, color='white', weight='bold')
        ax6.text(x, y-0.7, name, ha='center', va='top', fontsize=7, color=COLORS['text_secondary'])

    # IOC Correlation
    ax7 = fig.add_subplot(gs[2:, 2:])
    ax7.set_facecolor(COLORS['bg_card'])
    ax7.set_title('Threat Intelligence Matches', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)
    ax7.axis('off')

    iocs = [
        ('IP Address', '192.168.1.100', 'Cobalt Strike C2', COLORS['critical']),
        ('File Hash', 'a4f3bc...', 'Known Ransomware', COLORS['critical']),
        ('Domain', 'evil.com', 'Phishing Campaign', COLORS['danger']),
        ('Process', 'mimikatz.exe', 'Credential Theft', COLORS['danger']),
        ('Registry Key', 'Run\\Backdoor', 'Persistence', COLORS['warning']),
        ('Network Port', 'TCP:4444', 'Reverse Shell', COLORS['critical']),
    ]

    y_pos = 0.92
    for ioc_type, value, threat, color in iocs:
        # Background box
        rect = FancyBboxPatch((0.05, y_pos-0.12), 0.9, 0.13,
                              boxstyle="round,pad=0.01",
                              facecolor=COLORS['bg_card_hover'],
                              edgecolor=color,
                              linewidth=2,
                              alpha=0.6)
        ax7.add_patch(rect)

        ax7.text(0.08, y_pos-0.03, f'{ioc_type}:', fontsize=9, color=COLORS['text_secondary'], weight='bold')
        ax7.text(0.08, y_pos-0.08, value, fontsize=8, color=COLORS['text_primary'], family='monospace')
        ax7.text(0.92, y_pos-0.055, threat, ha='right', fontsize=9, color=color, weight='bold')

        y_pos -= 0.155

    plt.savefig('docs/images/dashboard-hunting.png', dpi=150, bbox_inches='tight', facecolor=COLORS['bg_dark'])
    print("✅ Generated: dashboard-hunting.png")
    plt.close()

def generate_dlp_dashboard():
    """Generate DLP Management Dashboard"""
    fig = plt.figure(figsize=(16, 10), facecolor=COLORS['bg_dark'])

    fig.suptitle('Data Loss Prevention - Policy Management',
                 fontsize=20, fontweight='bold', color=COLORS['text_primary'], y=0.98)
    fig.text(0.5, 0.95, 'Cryptographic fingerprinting protecting 2.4M sensitive documents',
             ha='center', fontsize=11, color=COLORS['text_secondary'])

    gs = fig.add_gridspec(4, 4, hspace=0.4, wspace=0.3, top=0.92, bottom=0.05, left=0.05, right=0.95)

    # Metrics Cards
    ax1 = fig.add_subplot(gs[0, 0])
    add_card(ax1, 'VIOLATIONS', '47', '↓ 23% vs last week', trend=-23)

    ax2 = fig.add_subplot(gs[0, 1])
    add_card(ax2, 'POLICIES', '18', 'active rules')

    ax3 = fig.add_subplot(gs[0, 2])
    add_card(ax3, 'FILES SCANNED', '2.4M', 'fingerprinted')

    ax4 = fig.add_subplot(gs[0, 3])
    add_card(ax4, 'BLOCKED', '12', 'exfiltration attempts')

    # Violations Trend
    ax5 = fig.add_subplot(gs[1, :])
    ax5.set_facecolor(COLORS['bg_card'])
    ax5.set_title('DLP Violations Trend (Last 30 Days)', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    days = np.arange(30)
    violations = np.maximum(0, 50 - days + np.random.normal(0, 5, 30))
    blocked = violations * 0.7 + np.random.normal(0, 2, 30)

    ax5.plot(days, violations, color=COLORS['danger'], linewidth=2.5, label='Violations Detected', marker='o', markersize=4)
    ax5.plot(days, blocked, color=COLORS['accent'], linewidth=2.5, label='Blocked', marker='s', markersize=4)
    ax5.fill_between(days, violations, alpha=0.2, color=COLORS['danger'])
    ax5.fill_between(days, blocked, alpha=0.2, color=COLORS['accent'])

    ax5.set_xlabel('Days Ago', fontsize=10, color=COLORS['text_secondary'])
    ax5.set_ylabel('Violation Count', fontsize=10, color=COLORS['text_secondary'])
    ax5.legend(loc='upper right', framealpha=0.9, facecolor=COLORS['bg_card_hover'])
    ax5.grid(True, alpha=0.2, linestyle='--')
    ax5.spines['top'].set_visible(False)
    ax5.spines['right'].set_visible(False)

    # Data Types Protected
    ax6 = fig.add_subplot(gs[2, :2])
    ax6.set_facecolor(COLORS['bg_card'])
    ax6.set_title('Protected Data Types', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    data_types = ['Credit Cards', 'SSN/PII', 'Source Code', 'Financial', 'Health Records', 'API Keys']
    files_count = [245000, 180000, 420000, 95000, 380000, 62000]

    bars = ax6.barh(data_types, files_count, color=COLORS['primary'], alpha=0.8, edgecolor=COLORS['border'], linewidth=1.5)

    for i, (bar, val) in enumerate(zip(bars, files_count)):
        ax6.text(val + 10000, i, f'{val/1000:.0f}K', va='center', fontsize=10, color=COLORS['text_primary'], weight='bold')

    ax6.set_xlabel('Files Fingerprinted', fontsize=10, color=COLORS['text_secondary'])
    ax6.grid(axis='x', alpha=0.2, linestyle='--')
    ax6.spines['top'].set_visible(False)
    ax6.spines['right'].set_visible(False)

    # Violation Channels
    ax7 = fig.add_subplot(gs[2, 2:])
    ax7.set_facecolor(COLORS['bg_card'])
    ax7.set_title('Violations by Channel', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    channels = ['Email', 'Cloud Upload', 'USB Drive', 'Print', 'Network Share']
    violations_ch = [18, 12, 8, 5, 4]
    channel_colors = [COLORS['danger'], COLORS['high'], COLORS['medium'], COLORS['warning'], COLORS['accent']]

    wedges, texts, autotexts = ax7.pie(violations_ch, labels=channels, autopct='%1.0f%%',
                                        colors=channel_colors, startangle=90,
                                        wedgeprops=dict(width=0.4, edgecolor=COLORS['border'], linewidth=2),
                                        textprops=dict(color=COLORS['text_primary'], fontsize=10, weight='bold'))

    # Active Policies
    ax8 = fig.add_subplot(gs[3, :])
    ax8.set_facecolor(COLORS['bg_card'])
    ax8.set_title('Active DLP Policies', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)
    ax8.axis('off')

    policies = [
        ('PCI DSS - Credit Card Protection', '245K files', 'High', COLORS['danger']),
        ('HIPAA - PHI Compliance', '380K files', 'High', COLORS['danger']),
        ('Source Code Protection', '420K files', 'Medium', COLORS['warning']),
        ('Customer PII - GDPR', '180K files', 'High', COLORS['danger']),
        ('API Keys & Credentials', '62K files', 'Critical', COLORS['critical']),
        ('Financial Reports', '95K files', 'Medium', COLORS['warning']),
    ]

    y_pos = 0.92
    for name, files, severity, color in policies:
        # Background box
        rect = FancyBboxPatch((0.03, y_pos-0.12), 0.94, 0.13,
                              boxstyle="round,pad=0.01",
                              facecolor=COLORS['bg_card_hover'],
                              edgecolor=COLORS['border'],
                              linewidth=1,
                              alpha=0.5)
        ax8.add_patch(rect)

        # Severity indicator
        circle = Circle((0.06, y_pos-0.055), 0.02, facecolor=color, edgecolor='none', alpha=0.9)
        ax8.add_patch(circle)

        ax8.text(0.10, y_pos-0.03, name, fontsize=10, color=COLORS['text_primary'], weight='bold')
        ax8.text(0.10, y_pos-0.08, files, fontsize=9, color=COLORS['text_secondary'])
        ax8.text(0.94, y_pos-0.055, severity, ha='right', fontsize=9, color=color, weight='bold')

        y_pos -= 0.155

    plt.savefig('docs/images/dashboard-dlp.png', dpi=150, bbox_inches='tight', facecolor=COLORS['bg_dark'])
    print("✅ Generated: dashboard-dlp.png")
    plt.close()

def generate_executive_dashboard():
    """Generate Executive Dashboard"""
    fig = plt.figure(figsize=(16, 10), facecolor=COLORS['bg_dark'])

    fig.suptitle('Executive Security Dashboard - At a Glance',
                 fontsize=20, fontweight='bold', color=COLORS['text_primary'], y=0.98)
    fig.text(0.5, 0.95, 'Enterprise-wide security posture and risk metrics',
             ha='center', fontsize=11, color=COLORS['text_secondary'])

    gs = fig.add_gridspec(4, 6, hspace=0.4, wspace=0.3, top=0.92, bottom=0.05, left=0.05, right=0.95)

    # Key Metrics
    ax1 = fig.add_subplot(gs[0, :2])
    add_card(ax1, 'RISK SCORE', '72/100', '↑ 5 pts (Good)', trend=5)

    ax2 = fig.add_subplot(gs[0, 2:4])
    add_card(ax2, 'COMPLIANCE', '98.7%', 'SOC2, HIPAA, GDPR')

    ax3 = fig.add_subplot(gs[0, 4:])
    add_card(ax3, 'COST SAVINGS', '$840K', 'vs legacy tools')

    # Security Posture Trend
    ax4 = fig.add_subplot(gs[1, :3])
    ax4.set_facecolor(COLORS['bg_card'])
    ax4.set_title('Security Posture Score (90 Days)', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    days = np.arange(90)
    score = 60 + (days * 0.12) + np.sin(days * 0.1) * 3

    ax4.plot(days, score, color=COLORS['accent'], linewidth=3, label='Posture Score')
    ax4.fill_between(days, score, alpha=0.3, color=COLORS['accent'])
    ax4.axhline(y=70, color=COLORS['warning'], linestyle='--', linewidth=1.5, label='Target', alpha=0.7)

    ax4.set_xlabel('Days', fontsize=10, color=COLORS['text_secondary'])
    ax4.set_ylabel('Score (0-100)', fontsize=10, color=COLORS['text_secondary'])
    ax4.legend(loc='lower right', framealpha=0.9, facecolor=COLORS['bg_card_hover'])
    ax4.grid(True, alpha=0.2, linestyle='--')
    ax4.spines['top'].set_visible(False)
    ax4.spines['right'].set_visible(False)
    ax4.set_ylim(50, 85)

    # Compliance Status
    ax5 = fig.add_subplot(gs[1, 3:])
    ax5.set_facecolor(COLORS['bg_card'])
    ax5.set_title('Compliance Framework Status', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    frameworks = ['SOC 2\nType II', 'HIPAA', 'GDPR', 'PCI DSS', 'ISO\n27001']
    compliance = [100, 99, 98, 97, 96]
    colors_comp = [COLORS['accent'] if c >= 98 else COLORS['warning'] for c in compliance]

    bars = ax5.bar(frameworks, compliance, color=colors_comp, alpha=0.8, edgecolor=COLORS['border'], linewidth=1.5, width=0.6)

    for bar, val in zip(bars, compliance):
        height = bar.get_height()
        ax5.text(bar.get_x() + bar.get_width()/2., height + 0.5,
                f'{val}%', ha='center', va='bottom', fontsize=10, color=COLORS['text_primary'], weight='bold')

    ax5.set_ylabel('Compliance %', fontsize=10, color=COLORS['text_secondary'])
    ax5.set_ylim(90, 105)
    ax5.axhline(y=95, color=COLORS['danger'], linestyle='--', linewidth=1.5, alpha=0.5)
    ax5.grid(axis='y', alpha=0.2, linestyle='--')
    ax5.spines['top'].set_visible(False)
    ax5.spines['right'].set_visible(False)

    # Threat Breakdown
    ax6 = fig.add_subplot(gs[2, :3])
    ax6.set_facecolor(COLORS['bg_card'])
    ax6.set_title('Threat Category Breakdown (30 Days)', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    categories = ['Malware', 'Phishing', 'DLP Violations', 'Insider Threat', 'Vuln Exploit', 'Misc']
    incidents = [34, 28, 47, 12, 8, 6]
    cat_colors = [COLORS['critical'], COLORS['danger'], COLORS['high'], COLORS['warning'], COLORS['medium'], COLORS['low']]

    wedges, texts, autotexts = ax6.pie(incidents, labels=categories, autopct='%1.0f%%',
                                        colors=cat_colors, startangle=90,
                                        wedgeprops=dict(edgecolor=COLORS['border'], linewidth=2),
                                        textprops=dict(color=COLORS['text_primary'], fontsize=9, weight='bold'))

    # ROI Metrics
    ax7 = fig.add_subplot(gs[2, 3:])
    ax7.set_facecolor(COLORS['bg_card'])
    ax7.set_title('Return on Investment (Year 1)', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)

    roi_cats = ['Tool\nConsolidation', 'Reduced\nHeadcount', 'Prevented\nBreaches', 'Compliance\nEfficiency']
    savings = [800, 200, 105, 90]

    bars2 = ax7.bar(roi_cats, savings, color=COLORS['accent'], alpha=0.8, edgecolor=COLORS['border'], linewidth=1.5, width=0.6)

    for bar, val in zip(bars2, savings):
        height = bar.get_height()
        ax7.text(bar.get_x() + bar.get_width()/2., height + 15,
                f'${val}K', ha='center', va='bottom', fontsize=10, color=COLORS['text_primary'], weight='bold')

    ax7.set_ylabel('Savings ($K)', fontsize=10, color=COLORS['text_secondary'])
    ax7.grid(axis='y', alpha=0.2, linestyle='--')
    ax7.spines['top'].set_visible(False)
    ax7.spines['right'].set_visible(False)

    # Risk Summary
    ax8 = fig.add_subplot(gs[3, :])
    ax8.set_facecolor(COLORS['bg_card'])
    ax8.set_title('Risk Summary & Recommendations', fontsize=12,
                  color=COLORS['text_primary'], weight='bold', pad=15)
    ax8.axis('off')

    risks = [
        ('✅', 'SOC Team Fully Operational', f'{COLORS["accent"]} All 6 analysts trained on Privé platform', COLORS['accent']),
        ('⚠️', 'Phishing Incidents +12% This Month', f'{COLORS["warning"]} Recommend additional user training', COLORS['warning']),
        ('✅', 'Zero Critical Vulnerabilities', f'{COLORS["accent"]} All endpoints patched within 48hrs', COLORS['accent']),
        ('⚠️', 'DLP Policy Coverage at 87%', f'{COLORS["warning"]} 13% of endpoints need policy updates', COLORS['warning']),
        ('✅', '$840K Annual Cost Savings Achieved', f'{COLORS["accent"]} ROI realized in 4.2 months', COLORS['accent']),
    ]

    y_pos = 0.88
    for icon, title, subtitle, color in risks:
        ax8.text(0.02, y_pos, icon, fontsize=16, va='center')
        ax8.text(0.06, y_pos+0.02, title, fontsize=11, color=COLORS['text_primary'], weight='bold', va='center')
        ax8.text(0.06, y_pos-0.05, subtitle, fontsize=9, color=color, va='center')
        y_pos -= 0.18

    plt.savefig('docs/images/dashboard-executive.png', dpi=150, bbox_inches='tight', facecolor=COLORS['bg_dark'])
    print("✅ Generated: dashboard-executive.png")
    plt.close()

if __name__ == '__main__':
    print("🎨 Generating professional dashboard mockups...")
    print("")

    generate_soc_dashboard()
    generate_threat_hunting_dashboard()
    generate_dlp_dashboard()
    generate_executive_dashboard()

    print("")
    print("✅ All dashboards generated successfully!")
    print("📁 Location: docs/images/")
