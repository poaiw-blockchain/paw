#!/usr/bin/env python3
"""
PAW Blockchain - Security Alerts System

Handles multi-channel alert routing for security findings:
- Email alerts for critical/high severity
- Slack notifications
-  issue creation
- PagerDuty integration (optional)
"""

import json
import os
import sys
import logging
import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Any
from enum import Enum
import subprocess
import requests
import yaml

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)


class Severity(Enum):
    """Security finding severity levels"""
    CRITICAL = 0
    HIGH = 1
    MEDIUM = 2
    LOW = 3
    INFO = 4


class AlertChannel(Enum):
    """Alert delivery channels"""
    EMAIL = "email"
    SLACK = "slack"
     = "issue_tracker"
    PAGERDUTY = "pagerduty"


class SecurityAlertSystem:
    """Manages security alert distribution"""

    def __init__(self, config_path: str = ".security/config.yml"):
        self.config_path = config_path
        self.config = self._load_config()
        self.findings: List[Dict[str, Any]] = []

    def _load_config(self) -> Dict[str, Any]:
        """Load security configuration"""
        config_file = Path(self.config_path)
        if not config_file.exists():
            logger.warning(f"Config not found: {self.config_path}")
            return {}

        try:
            with open(config_file, 'r') as f:
                return yaml.safe_load(f) or {}
        except Exception as e:
            logger.error(f"Failed to load config: {e}")
            return {}

    def load_findings(self, report_path: str = "security-report.json") -> bool:
        """Load security findings from report"""
        report_file = Path(report_path)
        if not report_file.exists():
            logger.error(f"Report not found: {report_path}")
            return False

        try:
            with open(report_file, 'r') as f:
                data = json.load(f)
                self.findings = data.get('findings', [])
                logger.info(f"Loaded {len(self.findings)} findings")
                return True
        except Exception as e:
            logger.error(f"Failed to load findings: {e}")
            return False

    def process_alerts(self) -> bool:
        """Process findings and send alerts"""
        if not self.findings:
            logger.info("No findings to alert on")
            return True

        logger.info(f"Processing {len(self.findings)} findings for alerts")

        # Group findings by severity
        grouped = self._group_by_severity()

        # Route alerts based on severity
        success = True
        for severity_level in [Severity.CRITICAL, Severity.HIGH, Severity.MEDIUM, Severity.LOW]:
            severity_str = severity_level.name
            if severity_str in grouped:
                findings = grouped[severity_str]
                channels = self._get_alert_channels(severity_str)

                for channel in channels:
                    try:
                        self._send_alert(channel, findings, severity_str)
                    except Exception as e:
                        logger.error(f"Failed to send {channel} alert: {e}")
                        success = False

        return success

    def _group_by_severity(self) -> Dict[str, List[Dict]]:
        """Group findings by severity level"""
        grouped = {}
        for finding in self.findings:
            severity = finding.get('severity', 'INFO')
            if severity not in grouped:
                grouped[severity] = []
            grouped[severity].append(finding)
        return grouped

    def _get_alert_channels(self, severity: str) -> List[AlertChannel]:
        """Determine which channels should receive alert"""
        alert_routing = self.config.get('alert_routing', {})
        channels_config = alert_routing.get(severity, {})
        channels_list = channels_config.get('channels', [])

        return [AlertChannel(ch) for ch in channels_list if ch in [c.value for c in AlertChannel]]

    def _send_alert(self, channel: AlertChannel, findings: List[Dict], severity: str):
        """Send alert via specified channel"""
        logger.info(f"Sending {severity} alert via {channel.value}")

        if channel == AlertChannel.EMAIL:
            self._send_email_alert(findings, severity)
        elif channel == AlertChannel.SLACK:
            self._send_slack_alert(findings, severity)
        elif channel == AlertChannel.:
            self._create_issue_tracker(findings, severity)
        elif channel == AlertChannel.PAGERDUTY:
            self._send_pagerduty_alert(findings, severity)

    def _send_email_alert(self, findings: List[Dict], severity: str):
        """Send email alert for security findings"""
        smtp_server = os.getenv('SECURITY_SMTP_SERVER', 'localhost')
        smtp_port = int(os.getenv('SECURITY_SMTP_PORT', '587'))
        sender_email = os.getenv('SECURITY_EMAIL_FROM', 'security@example.com')
        sender_password = os.getenv('SECURITY_EMAIL_PASSWORD', '')

        recipients = self.config.get('global', {}).get('notification_email', 'security-team@example.com')
        if isinstance(recipients, str):
            recipients = [recipients]

        try:
            msg = MIMEMultipart()
            msg['From'] = sender_email
            msg['To'] = ', '.join(recipients)
            msg['Subject'] = f"[{severity}] PAW Security Alert - {len(findings)} Finding(s)"

            # Build email body
            body = self._build_email_body(findings, severity)
            msg.attach(MIMEText(body, 'html'))

            # Send email
            with smtplib.SMTP(smtp_server, smtp_port) as server:
                server.starttls()
                if sender_password:
                    server.login(sender_email, sender_password)
                server.send_message(msg)

            logger.info(f"Email alert sent to {len(recipients)} recipients")

        except Exception as e:
            logger.error(f"Failed to send email: {e}")
            raise

    def _build_email_body(self, findings: List[Dict], severity: str) -> str:
        """Build HTML email body"""
        html_parts = [
            "<html><body>",
            f"<h2>PAW Blockchain Security Alert - {severity}</h2>",
            f"<p><strong>Timestamp:</strong> {datetime.now().isoformat()}</p>",
            f"<p><strong>Finding Count:</strong> {len(findings)}</p>",
            "<h3>Details</h3>",
            "<table border='1' cellpadding='10' style='border-collapse: collapse;'>",
            "<tr><th>Tool</th><th>Title</th><th>Description</th><th>Location</th></tr>"
        ]

        for finding in findings:
            html_parts.append(
                f"<tr>"
                f"<td>{finding.get('tool', 'Unknown')}</td>"
                f"<td>{finding.get('title', 'N/A')}</td>"
                f"<td>{finding.get('description', 'N/A')[:100]}...</td>"
                f"<td>{finding.get('location', 'N/A')}</td>"
                f"</tr>"
            )

        html_parts.extend([
            "</table>",
            "<p>Please review the full report in your security dashboard.</p>",
            "</body></html>"
        ])

        return "\n".join(html_parts)

    def _send_slack_alert(self, findings: List[Dict], severity: str):
        """Send Slack notification"""
        webhook_url = os.getenv('SECURITY_SLACK_WEBHOOK')
        if not webhook_url:
            logger.warning("Slack webhook not configured")
            return

        # Determine color based on severity
        color_map = {
            'CRITICAL': '#FF0000',
            'HIGH': '#FF6600',
            'MEDIUM': '#FFCC00',
            'LOW': '#00CC00',
            'INFO': '#0099FF'
        }

        # Build Slack message
        attachments = []
        for finding in findings[:10]:  # Limit to 10 findings per message
            attachments.append({
                "color": color_map.get(severity, '#999999'),
                "title": finding.get('title', 'Unknown'),
                "text": finding.get('description', 'No description'),
                "fields": [
                    {
                        "title": "Tool",
                        "value": finding.get('tool', 'Unknown'),
                        "short": True
                    },
                    {
                        "title": "Location",
                        "value": finding.get('location', 'Unknown'),
                        "short": True
                    }
                ]
            })

        payload = {
            "text": f"Security Alert: {severity} - {len(findings)} finding(s)",
            "attachments": attachments
        }

        try:
            response = requests.post(
                webhook_url,
                json=payload,
                timeout=10
            )
            response.raise_for_status()
            logger.info("Slack notification sent")
        except Exception as e:
            logger.error(f"Failed to send Slack alert: {e}")
            raise

    def _create_issue_tracker(self, findings: List[Dict], severity: str):
        """Create  issue for security findings"""
        tracker_token = os.getenv('_TOKEN')
        repo = self.config.get('global', {}).get('repository', '')

        if not tracker_token or not repo:
            logger.warning(" token or repository not configured")
            return

        try:
            headers = {
                'Authorization': f'token {tracker_token}',
                'Accept': 'application/vndhub.v3+json'
            }

            # Build issue body
            issue_body = f"## Security Alert: {severity}\n\n"
            issue_body += f"**Count:** {len(findings)}\n\n"
            issue_body += "### Findings\n\n"

            for finding in findings[:20]:  # Limit to 20 findings
                issue_body += f"- **{finding.get('tool', 'Unknown')}**: "
                issue_body += f"{finding.get('title', 'Unknown')} "
                issue_body += f"({finding.get('location', 'N/A')})\n"

            issue_data = {
                "title": f"[{severity}] Security Finding - {len(findings)} issue(s)",
                "body": issue_body,
                "labels": ["security", "vulnerability"]
            }

            # Create issue
            url = f"https://api.example.com/repos/{repo}/issues"
            response = requests.post(
                url,
                json=issue_data,
                headers=headers,
                timeout=10
            )

            if response.status_code == 201:
                issue_url = response.json().get('html_url', '')
                logger.info(f" issue created: {issue_url}")
            else:
                logger.error(f"Failed to create  issue: {response.status_code}")
                raise Exception(f" API error: {response.text}")

        except Exception as e:
            logger.error(f"Failed to create  issue: {e}")
            raise

    def _send_pagerduty_alert(self, findings: List[Dict], severity: str):
        """Send PagerDuty alert for critical findings"""
        pagerduty_key = os.getenv('PAGERDUTY_INTEGRATION_KEY')
        if not pagerduty_key:
            logger.warning("PagerDuty key not configured")
            return

        try:
            # Map severity to PagerDuty severity
            pd_severity = {
                'CRITICAL': 'critical',
                'HIGH': 'error',
                'MEDIUM': 'warning',
                'LOW': 'info',
                'INFO': 'info'
            }.get(severity, 'warning')

            event_data = {
                "routing_key": pagerduty_key,
                "event_action": "trigger",
                "payload": {
                    "summary": f"PAW Blockchain Security Alert: {severity} ({len(findings)} findings)",
                    "severity": pd_severity,
                    "source": "PAW Security Monitor",
                    "custom_details": {
                        "finding_count": len(findings),
                        "tools": list(set(f.get('tool', 'Unknown') for f in findings))
                    }
                }
            }

            response = requests.post(
                "https://events.pagerduty.com/v2/enqueue",
                json=event_data,
                timeout=10
            )
            response.raise_for_status()
            logger.info("PagerDuty alert sent")

        except Exception as e:
            logger.error(f"Failed to send PagerDuty alert: {e}")
            raise

    def generate_dashboard(self):
        """Generate security dashboard"""
        logger.info("Generating security dashboard...")

        dashboard_content = self._build_dashboard()

        dashboard_path = self.config.get('reporting', {}).get('dashboard', {}).get('path', 'SECURITY-DASHBOARD.md')
        with open(dashboard_path, 'w') as f:
            f.write(dashboard_content)

        logger.info(f"Dashboard generated: {dashboard_path}")

    def _build_dashboard(self) -> str:
        """Build security dashboard content"""
        lines = [
            "# PAW Blockchain Security Dashboard",
            f"\n**Last Updated:** {datetime.now().isoformat()}",
            f"\n**Total Findings:** {len(self.findings)}\n"
        ]

        # Count by severity
        by_severity = {}
        for finding in self.findings:
            severity = finding.get('severity', 'INFO')
            by_severity[severity] = by_severity.get(severity, 0) + 1

        lines.append("## Findings by Severity\n")
        for severity in ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO']:
            count = by_severity.get(severity, 0)
            emoji = {
                'CRITICAL': '🔴',
                'HIGH': '🟠',
                'MEDIUM': '🟡',
                'LOW': '🟢',
                'INFO': '🔵'
            }.get(severity, '⚪')
            lines.append(f"{emoji} **{severity}:** {count}")

        # Count by tool
        by_tool = {}
        for finding in self.findings:
            tool = finding.get('tool', 'Unknown')
            by_tool[tool] = by_tool.get(tool, 0) + 1

        lines.append("\n## Findings by Tool\n")
        for tool in sorted(by_tool.keys()):
            lines.append(f"- **{tool}:** {by_tool[tool]}")

        return "\n".join(lines)


def main():
    """Main entry point"""
    alert_system = SecurityAlertSystem()

    if not alert_system.load_findings():
        logger.error("Failed to load security findings")
        sys.exit(1)

    if not alert_system.process_alerts():
        logger.error("Alert processing encountered errors")
        sys.exit(1)

    alert_system.generate_dashboard()

    logger.info("Alert processing completed")
    sys.exit(0)


if __name__ == '__main__':
    main()
