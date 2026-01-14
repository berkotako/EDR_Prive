#!/usr/bin/env python3
"""Generate dashboard mockup images for Privé EDR/DLP platform."""

import matplotlib.pyplot as plt
import matplotlib.patches as patches
import numpy as np
from matplotlib.gridspec import GridSpec

# Set style
plt.style.use('seaborn-v0_8-darkgrid')
COLORS = {
    'primary': '#667eea',
    'secondary': '#764ba2',
    'critical': '#ef4444',
    'high': '#f59e0b',
    'medium': '#fbbf24',
    'low': '#10b981',
    'info': '#3b82f6',
    'bg': '#1f2937',
    'card': '#374151',
    'text': '#f3f4f6'
}

def create_soc_dashboard():
    """Create SOC Dashboard mockup."""
    fig = plt.figure(figsize=(16, 10), facecolor=COLORS['bg'])
    fig.suptitle('Privé - Security Operations Center', fontsize=20, color=COLORS['text'], weight='bold', y=0.98)

    gs = GridSpec(3, 3, figure=fig, hspace=0.3, wspace=0.3)

    # Metrics cards (top row)
    ax1 = fig.add_subplot(gs[0, 0])
    ax1.axis('off')
    ax1.add_patch(patches.Rectangle((0.1, 0.2), 0.8, 0.6, facecolor=COLORS['card'], edgecolor=COLORS['primary'], linewidth=2))
    ax1.text(0.5, 0.7, '12,547', ha='center', va='center', fontsize=24, weight='bold', color=COLORS['text'])
    ax1.text(0.5, 0.4, 'Events/Hour', ha='center', va='center', fontsize=12, color=COLORS['text'], alpha=0.7)
    ax1.text(0.5, 0.25, '▲ 15% from last hour', ha='center', va='center', fontsize=9, color=COLORS['low'])

    ax2 = fig.add_subplot(gs[0, 1])
    ax2.axis('off')
    ax2.add_patch(patches.Rectangle((0.1, 0.2), 0.8, 0.6, facecolor=COLORS['card'], edgecolor=COLORS['critical'], linewidth=2))
    ax2.text(0.5, 0.7, '23', ha='center', va='center', fontsize=24, weight='bold', color=COLORS['critical'])
    ax2.text(0.5, 0.4, 'Critical Alerts', ha='center', va='center', fontsize=12, color=COLORS['text'], alpha=0.7)
    ax2.text(0.5, 0.25, '▼ 8 from yesterday', ha='center', va='center', fontsize=9, color=COLORS['low'])

    ax3 = fig.add_subplot(gs[0, 2])
    ax3.axis('off')
    ax3.add_patch(patches.Rectangle((0.1, 0.2), 0.8, 0.6, facecolor=COLORS['card'], edgecolor=COLORS['info'], linewidth=2))
    ax3.text(0.5, 0.7, '1,247', ha='center', va='center', fontsize=24, weight='bold', color=COLORS['text'])
    ax3.text(0.5, 0.4, 'Active Endpoints', ha='center', va='center', fontsize=12, color=COLORS['text'], alpha=0.7)
    ax3.text(0.5, 0.25, '99.2% online', ha='center', va='center', fontsize=9, color=COLORS['low'])

    # Event timeline (middle left)
    ax4 = fig.add_subplot(gs[1, :2], facecolor=COLORS['card'])
    hours = np.arange(24)
    events = np.random.poisson(10000, 24) + np.sin(hours / 4) * 2000
    critical = np.random.poisson(20, 24)

    ax4.plot(hours, events, color=COLORS['primary'], linewidth=2, label='Total Events')
    ax4.fill_between(hours, events, alpha=0.3, color=COLORS['primary'])
    ax4_twin = ax4.twinx()
    ax4_twin.bar(hours, critical, alpha=0.6, color=COLORS['critical'], label='Critical', width=0.6)

    ax4.set_title('24-Hour Event Timeline', color=COLORS['text'], fontsize=14, weight='bold', pad=10)
    ax4.set_xlabel('Hour', color=COLORS['text'])
    ax4.set_ylabel('Total Events', color=COLORS['text'])
    ax4_twin.set_ylabel('Critical Alerts', color=COLORS['text'])
    ax4.tick_params(colors=COLORS['text'])
    ax4_twin.tick_params(colors=COLORS['text'])
    ax4.legend(loc='upper left', facecolor=COLORS['card'], edgecolor=COLORS['text'], labelcolor=COLORS['text'])
    ax4_twin.legend(loc='upper right', facecolor=COLORS['card'], edgecolor=COLORS['text'], labelcolor=COLORS['text'])
    ax4.grid(True, alpha=0.2, color=COLORS['text'])

    # MITRE ATT&CK heatmap (middle right)
    ax5 = fig.add_subplot(gs[1, 2], facecolor=COLORS['card'])
    tactics = ['Initial\nAccess', 'Execution', 'Persistence', 'Priv Esc', 'Defense\nEvasion', 'Credential\nAccess']
    values = np.random.randint(0, 50, 6)
    colors_heat = [COLORS['low'] if v < 10 else COLORS['medium'] if v < 25 else COLORS['high'] if v < 40 else COLORS['critical'] for v in values]

    bars = ax5.barh(tactics, values, color=colors_heat, edgecolor='white', linewidth=1.5)
    ax5.set_title('MITRE ATT&CK Coverage', color=COLORS['text'], fontsize=14, weight='bold', pad=10)
    ax5.set_xlabel('Detections (24h)', color=COLORS['text'])
    ax5.tick_params(colors=COLORS['text'])
    ax5.grid(True, alpha=0.2, color=COLORS['text'], axis='x')

    # Add value labels
    for i, (bar, value) in enumerate(zip(bars, values)):
        ax5.text(value + 1, i, str(value), va='center', color=COLORS['text'], weight='bold')

    # Severity distribution (bottom left)
    ax6 = fig.add_subplot(gs[2, 0], facecolor=COLORS['card'])
    severities = ['Info', 'Low', 'Medium', 'High', 'Critical']
    counts = [5420, 1230, 450, 180, 23]
    colors_sev = [COLORS['info'], COLORS['low'], COLORS['medium'], COLORS['high'], COLORS['critical']]

    wedges, texts, autotexts = ax6.pie(counts, labels=severities, autopct='%1.1f%%',
                                        colors=colors_sev, startangle=90, textprops={'color': COLORS['text']})
    for autotext in autotexts:
        autotext.set_color('white')
        autotext.set_weight('bold')
    ax6.set_title('Severity Distribution', color=COLORS['text'], fontsize=14, weight='bold', pad=10)

    # Top affected endpoints (bottom middle)
    ax7 = fig.add_subplot(gs[2, 1], facecolor=COLORS['card'])
    ax7.axis('off')
    ax7.text(0.5, 0.95, 'Top Affected Endpoints', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    endpoints = [
        ('WKS-Finance-042', 347, COLORS['critical']),
        ('SRV-Database-01', 289, COLORS['high']),
        ('WKS-HR-015', 156, COLORS['medium']),
        ('SRV-Web-Proxy', 123, COLORS['medium']),
        ('WKS-Dev-087', 98, COLORS['low'])
    ]

    y_pos = 0.78
    for name, count, color in endpoints:
        ax7.add_patch(patches.Rectangle((0.05, y_pos - 0.08), 0.9, 0.12, facecolor=COLORS['card'], edgecolor=color, linewidth=2))
        ax7.text(0.08, y_pos - 0.02, name, va='center', fontsize=10, color=COLORS['text'], weight='bold')
        ax7.text(0.92, y_pos - 0.02, str(count), ha='right', va='center', fontsize=11, color=color, weight='bold')
        y_pos -= 0.15

    # Recent critical alerts (bottom right)
    ax8 = fig.add_subplot(gs[2, 2], facecolor=COLORS['card'])
    ax8.axis('off')
    ax8.text(0.5, 0.95, 'Recent Critical Alerts', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    alerts = [
        ('Ransomware Activity', '2m ago'),
        ('Privilege Escalation', '5m ago'),
        ('Data Exfiltration', '8m ago'),
        ('Lateral Movement', '12m ago'),
        ('Suspicious PowerShell', '15m ago')
    ]

    y_pos = 0.78
    for alert, time in alerts:
        ax8.text(0.05, y_pos, f'● {alert}', va='center', fontsize=9, color=COLORS['critical'], weight='bold')
        ax8.text(0.95, y_pos, time, ha='right', va='center', fontsize=8, color=COLORS['text'], alpha=0.6)
        y_pos -= 0.15

    plt.savefig('/home/user/EDR_Prive/docs/images/dashboard-soc.png', dpi=150, bbox_inches='tight', facecolor=COLORS['bg'])
    print("✓ Generated: dashboard-soc.png")
    plt.close()

def create_hunting_dashboard():
    """Create Threat Hunting Dashboard mockup."""
    fig = plt.figure(figsize=(16, 10), facecolor=COLORS['bg'])
    fig.suptitle('Privé - Threat Hunting Workbench', fontsize=20, color=COLORS['text'], weight='bold', y=0.98)

    gs = GridSpec(3, 2, figure=fig, hspace=0.3, wspace=0.3)

    # Query builder (top)
    ax1 = fig.add_subplot(gs[0, :], facecolor=COLORS['card'])
    ax1.axis('off')
    ax1.add_patch(patches.Rectangle((0.02, 0.15), 0.96, 0.7, facecolor='#2d3748', edgecolor=COLORS['primary'], linewidth=2, alpha=0.5))
    ax1.text(0.03, 0.72, 'SELECT * FROM telemetry_events WHERE event_type = \'PROCESS_START\'',
             va='top', fontsize=11, color='#7dd3fc', family='monospace')
    ax1.text(0.03, 0.55, 'AND mitre_technique LIKE \'T1059%\'',
             va='top', fontsize=11, color='#7dd3fc', family='monospace')
    ax1.text(0.03, 0.38, 'AND timestamp >= now() - INTERVAL 24 HOUR',
             va='top', fontsize=11, color='#7dd3fc', family='monospace')
    ax1.text(0.03, 0.21, 'ORDER BY timestamp DESC LIMIT 100;',
             va='top', fontsize=11, color='#7dd3fc', family='monospace')

    ax1.add_patch(patches.Rectangle((0.82, 0.20), 0.15, 0.15, facecolor=COLORS['primary'], edgecolor='white', linewidth=1))
    ax1.text(0.895, 0.275, 'RUN QUERY', ha='center', va='center', fontsize=10, color='white', weight='bold')

    ax1.text(0.5, 0.92, 'Advanced Threat Hunt Query Builder', ha='center', va='center', fontsize=14, weight='bold', color=COLORS['text'])

    # Process tree visualization (middle left)
    ax2 = fig.add_subplot(gs[1, 0], facecolor=COLORS['card'])
    ax2.axis('off')
    ax2.text(0.5, 0.95, 'Process Execution Tree', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    # Draw tree structure
    processes = [
        (0.1, 0.80, 'explorer.exe (PID: 1234)', COLORS['low']),
        (0.2, 0.65, '└─ cmd.exe (PID: 2345)', COLORS['medium']),
        (0.3, 0.50, '   └─ powershell.exe (PID: 3456)', COLORS['high']),
        (0.4, 0.35, '      └─ mimikatz.exe (PID: 4567)', COLORS['critical']),
        (0.1, 0.20, 'svchost.exe (PID: 5678)', COLORS['low'])
    ]

    for x, y, text, color in processes:
        ax2.text(x, y, text, va='center', fontsize=10, color=color, family='monospace', weight='bold')

    # Timeline (middle right)
    ax3 = fig.add_subplot(gs[1, 1], facecolor=COLORS['card'])
    times = np.arange(0, 60, 5)
    events_timeline = np.random.randint(10, 100, len(times))

    colors_timeline = []
    for e in events_timeline:
        if e > 70:
            colors_timeline.append(COLORS['critical'])
        elif e > 50:
            colors_timeline.append(COLORS['high'])
        elif e > 30:
            colors_timeline.append(COLORS['medium'])
        else:
            colors_timeline.append(COLORS['low'])

    ax3.bar(times, events_timeline, color=colors_timeline, width=4, edgecolor='white', linewidth=0.5)
    ax3.set_title('Event Timeline (Last Hour)', color=COLORS['text'], fontsize=14, weight='bold', pad=10)
    ax3.set_xlabel('Minutes Ago', color=COLORS['text'])
    ax3.set_ylabel('Event Count', color=COLORS['text'])
    ax3.tick_params(colors=COLORS['text'])
    ax3.grid(True, alpha=0.2, color=COLORS['text'])

    # IOC Correlation (bottom left)
    ax4 = fig.add_subplot(gs[2, 0], facecolor=COLORS['card'])
    ax4.axis('off')
    ax4.text(0.5, 0.95, 'IOC Correlation Matrix', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    iocs = [
        ('Suspicious IP: 192.0.2.45', 'T1071 - C2', 12, COLORS['critical']),
        ('File Hash: c3ab8ff...', 'T1204 - Execution', 8, COLORS['high']),
        ('Domain: evil[.]com', 'T1566 - Phishing', 5, COLORS['medium']),
        ('Registry Key: HKLM\\...', 'T1547 - Persistence', 3, COLORS['low'])
    ]

    y_pos = 0.78
    for ioc, technique, count, color in iocs:
        ax4.add_patch(patches.Rectangle((0.05, y_pos - 0.10), 0.9, 0.15, facecolor=COLORS['card'], edgecolor=color, linewidth=2))
        ax4.text(0.08, y_pos - 0.02, ioc, va='center', fontsize=10, color=COLORS['text'], weight='bold')
        ax4.text(0.08, y_pos - 0.06, technique, va='center', fontsize=8, color=COLORS['text'], alpha=0.6)
        ax4.add_patch(patches.Circle((0.88, y_pos - 0.04), 0.03, facecolor=color, edgecolor='white', linewidth=1))
        ax4.text(0.88, y_pos - 0.04, str(count), ha='center', va='center', fontsize=9, color='white', weight='bold')
        y_pos -= 0.20

    # Query results table (bottom right)
    ax5 = fig.add_subplot(gs[2, 1], facecolor=COLORS['card'])
    ax5.axis('off')
    ax5.text(0.5, 0.95, 'Query Results (Top 5)', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    # Table header
    ax5.add_patch(patches.Rectangle((0.05, 0.82), 0.9, 0.08, facecolor=COLORS['primary'], edgecolor='white', linewidth=1))
    ax5.text(0.08, 0.86, 'Timestamp', va='center', fontsize=9, color='white', weight='bold')
    ax5.text(0.40, 0.86, 'Event', va='center', fontsize=9, color='white', weight='bold')
    ax5.text(0.85, 0.86, 'Severity', ha='center', va='center', fontsize=9, color='white', weight='bold')

    # Table rows
    results = [
        ('14:32:18', 'PowerShell encoded cmd', COLORS['critical']),
        ('14:29:45', 'Credential dumping', COLORS['critical']),
        ('14:25:12', 'Suspicious file write', COLORS['high']),
        ('14:20:33', 'Network scan detected', COLORS['medium']),
        ('14:18:09', 'Registry modification', COLORS['low'])
    ]

    y_pos = 0.74
    for time, event, color in results:
        ax5.add_patch(patches.Rectangle((0.05, y_pos - 0.05), 0.9, 0.08, facecolor='#2d3748', edgecolor=color, linewidth=1, alpha=0.3))
        ax5.text(0.08, y_pos - 0.01, time, va='center', fontsize=9, color=COLORS['text'], family='monospace')
        ax5.text(0.40, y_pos - 0.01, event, va='center', fontsize=9, color=COLORS['text'])
        ax5.add_patch(patches.Circle((0.85, y_pos - 0.01), 0.015, facecolor=color, edgecolor='white', linewidth=1))
        y_pos -= 0.12

    plt.savefig('/home/user/EDR_Prive/docs/images/dashboard-hunting.png', dpi=150, bbox_inches='tight', facecolor=COLORS['bg'])
    print("✓ Generated: dashboard-hunting.png")
    plt.close()

def create_dlp_dashboard():
    """Create DLP Dashboard mockup."""
    fig = plt.figure(figsize=(16, 10), facecolor=COLORS['bg'])
    fig.suptitle('Privé - Data Loss Prevention Management', fontsize=20, color=COLORS['text'], weight='bold', y=0.98)

    gs = GridSpec(3, 3, figure=fig, hspace=0.3, wspace=0.3)

    # DLP metrics (top row)
    ax1 = fig.add_subplot(gs[0, 0])
    ax1.axis('off')
    ax1.add_patch(patches.Rectangle((0.1, 0.2), 0.8, 0.6, facecolor=COLORS['card'], edgecolor=COLORS['critical'], linewidth=2))
    ax1.text(0.5, 0.7, '47', ha='center', va='center', fontsize=24, weight='bold', color=COLORS['critical'])
    ax1.text(0.5, 0.4, 'DLP Violations', ha='center', va='center', fontsize=12, color=COLORS['text'], alpha=0.7)
    ax1.text(0.5, 0.25, 'Last 24 hours', ha='center', va='center', fontsize=9, color=COLORS['text'], alpha=0.5)

    ax2 = fig.add_subplot(gs[0, 1])
    ax2.axis('off')
    ax2.add_patch(patches.Rectangle((0.1, 0.2), 0.8, 0.6, facecolor=COLORS['card'], edgecolor=COLORS['info'], linewidth=2))
    ax2.text(0.5, 0.7, '1.2TB', ha='center', va='center', fontsize=24, weight='bold', color=COLORS['text'])
    ax2.text(0.5, 0.4, 'Data Scanned', ha='center', va='center', fontsize=12, color=COLORS['text'], alpha=0.7)
    ax2.text(0.5, 0.25, '▲ 18% vs. last week', ha='center', va='center', fontsize=9, color=COLORS['low'])

    ax3 = fig.add_subplot(gs[0, 2])
    ax3.axis('off')
    ax3.add_patch(patches.Rectangle((0.1, 0.2), 0.8, 0.6, facecolor=COLORS['card'], edgecolor=COLORS['high'], linewidth=2))
    ax3.text(0.5, 0.7, '89', ha='center', va='center', fontsize=24, weight='bold', color=COLORS['high'])
    ax3.text(0.5, 0.4, 'Active Policies', ha='center', va='center', fontsize=12, color=COLORS['text'], alpha=0.7)
    ax3.text(0.5, 0.25, '12 modified today', ha='center', va='center', fontsize=9, color=COLORS['info'])

    # Violation trend (middle left/center)
    ax4 = fig.add_subplot(gs[1, :2], facecolor=COLORS['card'])
    days = np.arange(30)
    violations = np.random.poisson(40, 30) + np.sin(days / 5) * 15
    blocked = violations * 0.7 + np.random.randint(-5, 5, 30)

    ax4.plot(days, violations, color=COLORS['critical'], linewidth=2, marker='o', markersize=4, label='Total Violations')
    ax4.plot(days, blocked, color=COLORS['low'], linewidth=2, marker='s', markersize=4, label='Blocked')
    ax4.fill_between(days, blocked, alpha=0.3, color=COLORS['low'])

    ax4.set_title('DLP Violation Trend (30 Days)', color=COLORS['text'], fontsize=14, weight='bold', pad=10)
    ax4.set_xlabel('Days Ago', color=COLORS['text'])
    ax4.set_ylabel('Violations', color=COLORS['text'])
    ax4.tick_params(colors=COLORS['text'])
    ax4.legend(loc='upper left', facecolor=COLORS['card'], edgecolor=COLORS['text'], labelcolor=COLORS['text'])
    ax4.grid(True, alpha=0.2, color=COLORS['text'])

    # Top violated policies (middle right)
    ax5 = fig.add_subplot(gs[1, 2], facecolor=COLORS['card'])
    policies = ['SSN-US', 'CCN-All', 'HIPAA-PHI', 'PII-Email', 'Source\nCode']
    counts = [18, 12, 9, 5, 3]

    bars = ax5.barh(policies, counts, color=COLORS['critical'], edgecolor='white', linewidth=1.5)
    ax5.set_title('Top Violated Policies', color=COLORS['text'], fontsize=14, weight='bold', pad=10)
    ax5.set_xlabel('Violations (24h)', color=COLORS['text'])
    ax5.tick_params(colors=COLORS['text'])
    ax5.grid(True, alpha=0.2, color=COLORS['text'], axis='x')

    for i, (bar, count) in enumerate(zip(bars, counts)):
        ax5.text(count + 0.3, i, str(count), va='center', color=COLORS['text'], weight='bold')

    # Data classification (bottom left)
    ax6 = fig.add_subplot(gs[2, 0], facecolor=COLORS['card'])
    classifications = ['Public', 'Internal', 'Confidential', 'Restricted', 'Top Secret']
    data_counts = [45, 30, 15, 7, 3]
    colors_class = [COLORS['low'], COLORS['info'], COLORS['medium'], COLORS['high'], COLORS['critical']]

    wedges, texts, autotexts = ax6.pie(data_counts, labels=classifications, autopct='%1.1f%%',
                                        colors=colors_class, startangle=90, textprops={'color': COLORS['text']})
    for autotext in autotexts:
        autotext.set_color('white')
        autotext.set_weight('bold')
        autotext.set_fontsize(9)
    ax6.set_title('Data Classification', color=COLORS['text'], fontsize=14, weight='bold', pad=10)

    # Recent violations (bottom middle)
    ax7 = fig.add_subplot(gs[2, 1], facecolor=COLORS['card'])
    ax7.axis('off')
    ax7.text(0.5, 0.95, 'Recent Violations', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    violations_list = [
        ('SSN detected in email', 'john.doe@...', '2m ago', COLORS['critical']),
        ('Credit card in file', 'export.xlsx', '5m ago', COLORS['critical']),
        ('PHI in chat message', 'slack://...', '8m ago', COLORS['high']),
        ('Source code upload', 'github.com/...', '12m ago', COLORS['medium']),
        ('Customer list export', 'customers.csv', '18m ago', COLORS['high'])
    ]

    y_pos = 0.80
    for violation, location, time, color in violations_list:
        ax7.add_patch(patches.Rectangle((0.03, y_pos - 0.08), 0.94, 0.12, facecolor=COLORS['card'], edgecolor=color, linewidth=2))
        ax7.text(0.05, y_pos - 0.02, violation, va='center', fontsize=9, color=COLORS['text'], weight='bold')
        ax7.text(0.05, y_pos - 0.06, location, va='center', fontsize=8, color=COLORS['text'], alpha=0.6, style='italic')
        ax7.text(0.95, y_pos - 0.04, time, ha='right', va='center', fontsize=8, color=color)
        y_pos -= 0.16

    # Policy effectiveness (bottom right)
    ax8 = fig.add_subplot(gs[2, 2], facecolor=COLORS['card'])
    policy_types = ['Detection', 'Block', 'Encrypt', 'Alert', 'Audit']
    effectiveness = [95, 88, 92, 97, 100]
    colors_eff = [COLORS['low'] if e > 90 else COLORS['medium'] if e > 75 else COLORS['high'] for e in effectiveness]

    bars = ax8.bar(policy_types, effectiveness, color=colors_eff, edgecolor='white', linewidth=1.5)
    ax8.set_title('Policy Effectiveness', color=COLORS['text'], fontsize=14, weight='bold', pad=10)
    ax8.set_ylabel('Success Rate (%)', color=COLORS['text'])
    ax8.tick_params(colors=COLORS['text'])
    ax8.set_ylim(0, 100)
    ax8.grid(True, alpha=0.2, color=COLORS['text'], axis='y')
    ax8.axhline(y=90, color=COLORS['low'], linestyle='--', alpha=0.5, linewidth=1)

    for bar, value in zip(bars, effectiveness):
        ax8.text(bar.get_x() + bar.get_width() / 2, value + 2, f'{value}%',
                ha='center', va='bottom', color=COLORS['text'], weight='bold', fontsize=9)

    plt.savefig('/home/user/EDR_Prive/docs/images/dashboard-dlp.png', dpi=150, bbox_inches='tight', facecolor=COLORS['bg'])
    print("✓ Generated: dashboard-dlp.png")
    plt.close()

def create_executive_dashboard():
    """Create Executive Dashboard mockup."""
    fig = plt.figure(figsize=(16, 10), facecolor=COLORS['bg'])
    fig.suptitle('Privé - Executive Security Dashboard', fontsize=20, color=COLORS['text'], weight='bold', y=0.98)

    gs = GridSpec(3, 3, figure=fig, hspace=0.3, wspace=0.3)

    # Security score (top left, larger)
    ax1 = fig.add_subplot(gs[0:2, 0], facecolor=COLORS['card'])
    ax1.axis('off')

    # Draw circular progress
    theta = np.linspace(0, 2 * np.pi * 0.87, 100)
    r = 0.35
    x = 0.5 + r * np.cos(theta - np.pi / 2)
    y = 0.5 + r * np.sin(theta - np.pi / 2)
    ax1.plot(x, y, color=COLORS['low'], linewidth=15)

    # Background circle
    theta_bg = np.linspace(0, 2 * np.pi, 100)
    x_bg = 0.5 + r * np.cos(theta_bg - np.pi / 2)
    y_bg = 0.5 + r * np.sin(theta_bg - np.pi / 2)
    ax1.plot(x_bg, y_bg, color='#4b5563', linewidth=15, alpha=0.3)

    ax1.text(0.5, 0.55, '87', ha='center', va='center', fontsize=48, weight='bold', color=COLORS['low'])
    ax1.text(0.5, 0.40, 'Security Score', ha='center', va='center', fontsize=14, color=COLORS['text'])
    ax1.text(0.5, 0.32, '▲ 5 points this month', ha='center', va='center', fontsize=10, color=COLORS['low'])
    ax1.text(0.5, 0.95, 'Overall Posture', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    # Risk trend (top middle/right)
    ax2 = fig.add_subplot(gs[0, 1:], facecolor=COLORS['card'])
    months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun']
    critical_risk = [45, 38, 32, 28, 25, 23]
    high_risk = [120, 105, 95, 88, 82, 78]
    medium_risk = [280, 265, 250, 245, 240, 235]

    ax2.plot(months, critical_risk, color=COLORS['critical'], linewidth=3, marker='o', markersize=8, label='Critical')
    ax2.plot(months, high_risk, color=COLORS['high'], linewidth=3, marker='s', markersize=8, label='High')
    ax2.plot(months, medium_risk, color=COLORS['medium'], linewidth=3, marker='^', markersize=8, label='Medium')

    ax2.set_title('Risk Reduction Trend', color=COLORS['text'], fontsize=14, weight='bold', pad=10)
    ax2.set_ylabel('Open Risks', color=COLORS['text'])
    ax2.tick_params(colors=COLORS['text'])
    ax2.legend(loc='upper right', facecolor=COLORS['card'], edgecolor=COLORS['text'], labelcolor=COLORS['text'])
    ax2.grid(True, alpha=0.2, color=COLORS['text'])

    # Compliance status (middle middle/right)
    ax3 = fig.add_subplot(gs[1, 1:], facecolor=COLORS['card'])
    ax3.axis('off')
    ax3.text(0.5, 0.95, 'Compliance Framework Status', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    compliance = [
        ('GDPR', 98, COLORS['low']),
        ('HIPAA', 95, COLORS['low']),
        ('SOC 2', 87, COLORS['medium']),
        ('ISO 27001', 82, COLORS['medium']),
        ('PCI DSS', 91, COLORS['low'])
    ]

    y_pos = 0.80
    for framework, score, color in compliance:
        # Background bar
        ax3.add_patch(patches.Rectangle((0.25, y_pos - 0.05), 0.65, 0.08, facecolor='#4b5563', alpha=0.3))
        # Progress bar
        ax3.add_patch(patches.Rectangle((0.25, y_pos - 0.05), 0.65 * (score / 100), 0.08, facecolor=color))
        # Labels
        ax3.text(0.05, y_pos - 0.01, framework, va='center', fontsize=11, color=COLORS['text'], weight='bold')
        ax3.text(0.93, y_pos - 0.01, f'{score}%', ha='right', va='center', fontsize=11, color=color, weight='bold')
        y_pos -= 0.16

    # Monthly threat summary (bottom left)
    ax4 = fig.add_subplot(gs[2, 0], facecolor=COLORS['card'])
    categories = ['Malware', 'Phishing', 'Intrusion', 'DLP', 'Insider']
    incidents = [12, 28, 5, 47, 8]
    colors_cat = [COLORS['critical'], COLORS['high'], COLORS['medium'], COLORS['high'], COLORS['medium']]

    bars = ax4.bar(categories, incidents, color=colors_cat, edgecolor='white', linewidth=1.5)
    ax4.set_title('Incidents by Category', color=COLORS['text'], fontsize=14, weight='bold', pad=10)
    ax4.set_ylabel('Count (Last 30 Days)', color=COLORS['text'])
    ax4.tick_params(colors=COLORS['text'])
    ax4.grid(True, alpha=0.2, color=COLORS['text'], axis='y')

    for bar, count in zip(bars, incidents):
        ax4.text(bar.get_x() + bar.get_width() / 2, count + 1, str(count),
                ha='center', va='bottom', color=COLORS['text'], weight='bold', fontsize=10)

    # Mean time to respond (bottom middle)
    ax5 = fig.add_subplot(gs[2, 1], facecolor=COLORS['card'])
    ax5.axis('off')
    ax5.text(0.5, 0.95, 'Response Metrics', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    metrics = [
        ('Mean Time to Detect', '4.2', 'minutes', COLORS['low']),
        ('Mean Time to Respond', '12.8', 'minutes', COLORS['medium']),
        ('Mean Time to Contain', '35', 'minutes', COLORS['high']),
        ('Mean Time to Recover', '2.1', 'hours', COLORS['medium'])
    ]

    y_pos = 0.78
    for label, value, unit, color in metrics:
        ax5.add_patch(patches.Rectangle((0.05, y_pos - 0.08), 0.9, 0.12, facecolor=COLORS['card'], edgecolor=color, linewidth=2))
        ax5.text(0.08, y_pos - 0.02, label, va='center', fontsize=10, color=COLORS['text'])
        ax5.text(0.85, y_pos - 0.02, value, ha='right', va='center', fontsize=12, color=color, weight='bold')
        ax5.text(0.92, y_pos - 0.02, unit, ha='right', va='center', fontsize=8, color=COLORS['text'], alpha=0.6)
        y_pos -= 0.20

    # Cost savings (bottom right)
    ax6 = fig.add_subplot(gs[2, 2], facecolor=COLORS['card'])
    ax6.axis('off')
    ax6.text(0.5, 0.95, 'Security ROI', ha='center', va='top', fontsize=14, weight='bold', color=COLORS['text'])

    ax6.add_patch(patches.Rectangle((0.1, 0.50), 0.8, 0.30, facecolor=COLORS['primary'], edgecolor='white', linewidth=2))
    ax6.text(0.5, 0.68, '$2.4M', ha='center', va='center', fontsize=24, weight='bold', color='white')
    ax6.text(0.5, 0.57, 'Prevented Losses (YTD)', ha='center', va='center', fontsize=10, color='white')

    ax6.add_patch(patches.Rectangle((0.1, 0.15), 0.8, 0.25, facecolor=COLORS['card'], edgecolor=COLORS['low'], linewidth=2))
    ax6.text(0.5, 0.32, '47', ha='center', va='center', fontsize=20, weight='bold', color=COLORS['low'])
    ax6.text(0.5, 0.22, 'Incidents Blocked', ha='center', va='center', fontsize=10, color=COLORS['text'])

    plt.savefig('/home/user/EDR_Prive/docs/images/dashboard-executive.png', dpi=150, bbox_inches='tight', facecolor=COLORS['bg'])
    print("✓ Generated: dashboard-executive.png")
    plt.close()

if __name__ == '__main__':
    print("Generating Privé dashboard mockups...")
    print()
    create_soc_dashboard()
    create_hunting_dashboard()
    create_dlp_dashboard()
    create_executive_dashboard()
    print()
    print("✓ All dashboard images generated successfully!")
