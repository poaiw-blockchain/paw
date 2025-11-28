#!/usr/bin/env python3
"""
PAW Blockchain - Continuous Security Monitoring System (Python Component)

Automated security scanning for Python code, including:
- Bandit: Security issue detection
- Safety: Dependency vulnerability checking
- pip-audit: Package vulnerability auditing
- Semgrep: Static analysis and SAST
"""

import json
import os
import sys
import subprocess
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, asdict
from enum import Enum
import yaml

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class Severity(Enum):
    """Security finding severity levels"""
    CRITICAL = "CRITICAL"
    HIGH = "HIGH"
    MEDIUM = "MEDIUM"
    LOW = "LOW"
    INFO = "INFO"

    def __lt__(self, other):
        severity_order = {
            "CRITICAL": 0,
            "HIGH": 1,
            "MEDIUM": 2,
            "LOW": 3,
            "INFO": 4
        }
        return severity_order[self.value] < severity_order[other.value]


@dataclass
class SecurityFinding:
    """Represents a security finding"""
    tool: str
    severity: str
    title: str
    description: str
    location: str
    cwe: Optional[str] = None
    remediation: Optional[str] = None
    timestamp: Optional[str] = None

    def to_dict(self) -> Dict:
        return asdict(self)


class SecurityScanner:
    """Main security scanning orchestrator"""

    def __init__(self, config_path: str = ".security/config.yml"):
        self.config_path = config_path
        self.config = self._load_config()
        self.findings: List[SecurityFinding] = []
        self.scan_timestamp = datetime.now().isoformat()
        self.project_root = Path(__file__).parent.parent

    def _load_config(self) -> Dict[str, Any]:
        """Load security configuration from YAML"""
        config_file = Path(self.config_path)
        if not config_file.exists():
            logger.warning(f"Config file not found: {self.config_path}, using defaults")
            return {}

        try:
            with open(config_file, 'r') as f:
                return yaml.safe_load(f) or {}
        except Exception as e:
            logger.error(f"Failed to load config: {e}")
            return {}

    def run_all_scans(self) -> bool:
        """Run all enabled security scanners"""
        logger.info("Starting comprehensive Python security scanning...")

        python_config = self.config.get('python_security', {})
        tools = python_config.get('tools', {})

        all_passed = True

        if tools.get('bandit', {}).get('enabled', True):
            all_passed &= self._run_bandit(tools.get('bandit', {}))

        if tools.get('safety', {}).get('enabled', True):
            all_passed &= self._run_safety(tools.get('safety', {}))

        if tools.get('pip_audit', {}).get('enabled', True):
            all_passed &= self._run_pip_audit(tools.get('pip_audit', {}))

        if tools.get('semgrep', {}).get('enabled', True):
            all_passed &= self._run_semgrep(tools.get('semgrep', {}))

        # Generate report
        self._generate_report()

        return all_passed

    def _run_bandit(self, config: Dict) -> bool:
        """Run Bandit security scanner"""
        logger.info("Running Bandit security scanner...")
        try:
            cmd = [
                'bandit',
                '-r', 'src/',
                '-f', 'json',
                '-o', 'bandit-report.json'
            ]

            severity = config.get('severity', 'medium')
            if severity:
                cmd.extend(['-ll'] if severity == 'low' else [])

            result = subprocess.run(cmd, capture_output=True, text=True)

            # Parse results
            if os.path.exists('bandit-report.json'):
                with open('bandit-report.json', 'r') as f:
                    data = json.load(f)
                    for issue in data.get('results', []):
                        finding = SecurityFinding(
                            tool='bandit',
                            severity=issue.get('severity', 'MEDIUM'),
                            title=issue.get('test_name', 'Unknown'),
                            description=issue.get('issue_text', ''),
                            location=f"{issue.get('filename')}:{issue.get('line_number')}",
                            cwe=f"CWE-{issue.get('test_id', 'Unknown')}",
                            timestamp=self.scan_timestamp
                        )
                        self.findings.append(finding)

            # Check if we should fail
            fail_on_issues = config.get('fail_on_issues', True)
            has_findings = len(self.findings) > 0

            if fail_on_issues and has_findings:
                logger.warning(f"Bandit found {len(self.findings)} security issues")
                return False

            logger.info("Bandit scan completed successfully")
            return True

        except FileNotFoundError:
            logger.error("Bandit not installed. Install with: pip install bandit")
            return False
        except Exception as e:
            logger.error(f"Bandit scan failed: {e}")
            return False

    def _run_safety(self, config: Dict) -> bool:
        """Run Safety dependency vulnerability checker"""
        logger.info("Running Safety vulnerability check...")
        try:
            cmd = ['safety', 'check', '--json']

            result = subprocess.run(cmd, capture_output=True, text=True)

            if result.returncode != 0:
                try:
                    data = json.loads(result.stdout)
                    for vuln in data:
                        if isinstance(vuln, dict):
                            finding = SecurityFinding(
                                tool='safety',
                                severity='HIGH',  # Safety reports are usually high priority
                                title=vuln.get('package', 'Unknown'),
                                description=vuln.get('vulnerability', ''),
                                location='dependencies',
                                cwe=None,
                                timestamp=self.scan_timestamp
                            )
                            self.findings.append(finding)
                except json.JSONDecodeError:
                    pass

            fail_on_vuln = config.get('fail_on_vulnerability', True)
            if fail_on_vuln and len(self.findings) > 0:
                logger.warning("Safety found vulnerabilities in dependencies")
                return False

            logger.info("Safety check completed")
            return True

        except FileNotFoundError:
            logger.error("Safety not installed. Install with: pip install safety")
            return False
        except Exception as e:
            logger.error(f"Safety check failed: {e}")
            return False

    def _run_pip_audit(self, config: Dict) -> bool:
        """Run pip-audit for package vulnerability scanning"""
        logger.info("Running pip-audit...")
        try:
            cmd = ['pip-audit', '--format', 'json']

            result = subprocess.run(cmd, capture_output=True, text=True)

            if result.returncode != 0:
                try:
                    data = json.loads(result.stdout)
                    for vuln in data.get('vulnerabilities', []):
                        finding = SecurityFinding(
                            tool='pip-audit',
                            severity='HIGH',
                            title=vuln.get('name', 'Unknown'),
                            description=vuln.get('description', ''),
                            location='dependencies',
                            cwe=None,
                            remediation=vuln.get('fix_version'),
                            timestamp=self.scan_timestamp
                        )
                        self.findings.append(finding)
                except json.JSONDecodeError:
                    pass

            fail_on_vuln = config.get('fail_on_vulnerability', True)
            if fail_on_vuln and len(self.findings) > 0:
                logger.warning("pip-audit found vulnerabilities")
                return False

            logger.info("pip-audit completed")
            return True

        except FileNotFoundError:
            logger.error("pip-audit not installed. Install with: pip install pip-audit")
            return False
        except Exception as e:
            logger.error(f"pip-audit failed: {e}")
            return False

    def _run_semgrep(self, config: Dict) -> bool:
        """Run Semgrep static analysis"""
        logger.info("Running Semgrep analysis...")
        try:
            cmd = [
                'semgrep',
                '--config', config.get('config', 'p/security-audit'),
                '--json',
                '--output', 'semgrep-report.json',
                'src/'
            ]

            result = subprocess.run(cmd, capture_output=True, text=True)

            if os.path.exists('semgrep-report.json'):
                with open('semgrep-report.json', 'r') as f:
                    data = json.load(f)
                    for result_item in data.get('results', []):
                        severity = result_item.get('extra', {}).get('severity', 'MEDIUM')
                        finding = SecurityFinding(
                            tool='semgrep',
                            severity=severity,
                            title=result_item.get('check_id', 'Unknown'),
                            description=result_item.get('extra', {}).get('message', ''),
                            location=f"{result_item.get('path')}:{result_item.get('start', {}).get('line')}",
                            cwe=None,
                            timestamp=self.scan_timestamp
                        )
                        self.findings.append(finding)

            fail_on_findings = config.get('fail_on_findings', True)
            critical_findings = [f for f in self.findings if f.tool == 'semgrep' and f.severity in ['CRITICAL', 'HIGH']]

            if fail_on_findings and critical_findings:
                logger.warning(f"Semgrep found {len(critical_findings)} critical/high findings")
                return False

            logger.info("Semgrep analysis completed")
            return True

        except FileNotFoundError:
            logger.error("Semgrep not installed. Install with: pip install semgrep")
            return False
        except Exception as e:
            logger.error(f"Semgrep analysis failed: {e}")
            return False

    def _generate_report(self):
        """Generate security report"""
        report_data = {
            'scan_timestamp': self.scan_timestamp,
            'total_findings': len(self.findings),
            'by_severity': self._count_by_severity(),
            'by_tool': self._count_by_tool(),
            'findings': [f.to_dict() for f in self.findings]
        }

        # Save JSON report
        with open('security-report.json', 'w') as f:
            json.dump(report_data, f, indent=2)

        # Generate markdown report
        self._generate_markdown_report(report_data)

        logger.info(f"Security report generated with {len(self.findings)} findings")

    def _count_by_severity(self) -> Dict[str, int]:
        """Count findings by severity"""
        counts = {s.value: 0 for s in Severity}
        for finding in self.findings:
            if finding.severity in counts:
                counts[finding.severity] += 1
        return counts

    def _count_by_tool(self) -> Dict[str, int]:
        """Count findings by tool"""
        counts = {}
        for finding in self.findings:
            counts[finding.tool] = counts.get(finding.tool, 0) + 1
        return counts

    def _generate_markdown_report(self, data: Dict):
        """Generate markdown security report"""
        lines = [
            "# Python Security Scan Report",
            f"\n**Scan Timestamp:** {data['scan_timestamp']}",
            f"\n**Total Findings:** {data['total_findings']}\n"
        ]

        # Summary by severity
        lines.append("## Summary by Severity\n")
        for severity, count in data['by_severity'].items():
            if count > 0:
                lines.append(f"- **{severity}:** {count}")

        # Summary by tool
        lines.append("\n## Summary by Tool\n")
        for tool, count in data['by_tool'].items():
            lines.append(f"- **{tool}:** {count}")

        # Detailed findings
        if data['findings']:
            lines.append("\n## Detailed Findings\n")
            for finding in data['findings']:
                lines.append(f"### {finding['title']}")
                lines.append(f"- **Tool:** {finding['tool']}")
                lines.append(f"- **Severity:** {finding['severity']}")
                lines.append(f"- **Location:** {finding['location']}")
                if finding.get('description'):
                    lines.append(f"- **Description:** {finding['description']}")
                if finding.get('remediation'):
                    lines.append(f"- **Remediation:** {finding['remediation']}")
                lines.append("")

        with open('security-report.md', 'w') as f:
            f.write('\n'.join(lines))

    def export_sarif(self):
        """Export findings in SARIF format for """
        sarif = {
            "version": "2.1.0",
            "runs": [{
                "tool": {
                    "driver": {
                        "name": "PAW Security Scanner",
                        "version": "1.0.0",
                        "informationUri": "https://example.com/paw-org/paw"
                    }
                },
                "results": []
            }]
        }

        for finding in self.findings:
            result = {
                "ruleId": finding.tool,
                "level": finding.severity.lower(),
                "message": {
                    "text": finding.title
                },
                "locations": [{
                    "physicalLocation": {
                        "artifactLocation": {
                            "uri": finding.location.split(':')[0]
                        },
                        "region": {
                            "startLine": int(finding.location.split(':')[1]) if ':' in finding.location else 1
                        }
                    }
                }]
            }
            sarif["runs"][0]["results"].append(result)

        with open('security-report.sarif', 'w') as f:
            json.dump(sarif, f, indent=2)

        logger.info("SARIF report generated")


def main():
    """Main entry point"""
    scanner = SecurityScanner()
    success = scanner.run_all_scans()
    scanner.export_sarif()

    # Exit with appropriate code
    sys.exit(0 if success else 1)


if __name__ == '__main__':
    main()
