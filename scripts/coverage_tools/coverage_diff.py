#!/usr/bin/env python3
"""
Compare Go and Python coverage metrics.

Analyzes coverage reports from both Go and Python projects,
identifies gaps, generates comparative reports, and tracks progress.

Usage:
    python coverage_diff.py --go-coverage coverage.out --py-coverage coverage.json
    python coverage_diff.py --compare baseline
    python coverage_diff.py --report --output report.html

Examples:
    python coverage_diff.py
    python coverage_diff.py --go-coverage coverage.out --py-coverage coverage.json --show-gaps
    python coverage_diff.py --compare main --output comparison.txt
"""

import json
import sys
import subprocess
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass
from collections import defaultdict
import argparse
import re


@dataclass
class CoverageStats:
    """Coverage statistics."""
    language: str
    total_files: int
    covered_files: int
    total_coverage: float
    line_coverage: float
    branch_coverage: float
    statements: int
    covered_statements: int
    uncovered_statements: int
    modules: Dict[str, float]  # module -> coverage %


class GoGovernanceCoverageAnalyzer:
    """Analyze Go code coverage."""

    def parse_coverage_out(self, coverage_file: Path) -> Dict[str, float]:
        """Parse go coverage .out file."""
        coverage = {}

        try:
            with open(coverage_file, 'r') as f:
                for line in f:
                    if line.startswith('mode:'):
                        continue
                    parts = line.strip().split()
                    if len(parts) >= 3:
                        module = parts[0]
                        count = int(parts[2]) if parts[2].isdigit() else 0
                        if module not in coverage:
                            coverage[module] = {'covered': 0, 'total': 0}
                        coverage[module]['total'] += 1
                        if count > 0:
                            coverage[module]['covered'] += 1
        except Exception as e:
            print(f"Error parsing Go coverage: {e}", file=sys.stderr)

        return coverage

    def get_coverage_stats(self, coverage_file: Path) -> Optional[CoverageStats]:
        """Get Go coverage statistics."""
        coverage = self.parse_coverage_out(coverage_file)

        if not coverage:
            return None

        total_covered = sum(m['covered'] for m in coverage.values())
        total_statements = sum(m['total'] for m in coverage.values())

        if total_statements == 0:
            return None

        coverage_pct = (total_covered / total_statements) * 100

        # Group by module
        modules = {}
        for module_path, stats in coverage.items():
            module_name = module_path.split('/')[0] if '/' in module_path else module_path
            if module_name not in modules:
                modules[module_name] = {'covered': 0, 'total': 0}
            modules[module_name]['covered'] += stats['covered']
            modules[module_name]['total'] += stats['total']

        module_coverage = {
            name: (stats['covered'] / stats['total'] * 100) if stats['total'] > 0 else 0
            for name, stats in modules.items()
        }

        return CoverageStats(
            language='Go',
            total_files=len(coverage),
            covered_files=len([m for m in coverage.values() if m['covered'] > 0]),
            total_coverage=coverage_pct,
            line_coverage=coverage_pct,
            branch_coverage=0.0,  # Go coverage doesn't track branch coverage
            statements=total_statements,
            covered_statements=total_covered,
            uncovered_statements=total_statements - total_covered,
            modules=module_coverage,
        )


class PythonCoverageAnalyzer:
    """Analyze Python code coverage."""

    def parse_coverage_json(self, coverage_file: Path) -> Optional[CoverageStats]:
        """Parse Python coverage.json file."""
        try:
            with open(coverage_file, 'r') as f:
                data = json.load(f)

            totals = data.get('totals', {})
            files = data.get('files', {})

            coverage_pct = totals.get('percent_covered', 0.0)
            statements = totals.get('num_statements', 0)
            executed = totals.get('executed_lines', 0)

            # Group by module
            modules = defaultdict(lambda: {'covered': 0, 'total': 0})
            for file_path, file_data in files.items():
                # Extract module name
                parts = file_path.replace('\\', '/').split('/')
                if 'src' in parts:
                    idx = parts.index('src')
                    module = parts[idx + 1] if idx + 1 < len(parts) else 'unknown'
                else:
                    module = parts[0] if parts else 'unknown'

                file_summary = file_data.get('summary', {})
                file_covered = file_summary.get('executed_lines', 0)
                file_total = file_summary.get('num_statements', 0)

                modules[module]['covered'] += file_covered
                modules[module]['total'] += file_total

            module_coverage = {
                name: (stats['covered'] / stats['total'] * 100) if stats['total'] > 0 else 0
                for name, stats in modules.items()
            }

            return CoverageStats(
                language='Python',
                total_files=len(files),
                covered_files=len([f for f, d in files.items() if d.get('summary', {}).get('executed_lines', 0) > 0]),
                total_coverage=coverage_pct,
                line_coverage=coverage_pct,
                branch_coverage=totals.get('percent_covered_with_branch', 0.0),
                statements=statements,
                covered_statements=executed,
                uncovered_statements=statements - executed,
                modules=module_coverage,
            )
        except Exception as e:
            print(f"Error parsing Python coverage: {e}", file=sys.stderr)
            return None


class CoverageDiffAnalyzer:
    """Analyze coverage differences."""

    def __init__(self):
        self.go_analyzer = GoGovernanceCoverageAnalyzer()
        self.py_analyzer = PythonCoverageAnalyzer()

    def compare_coverage(self, go_file: Path, py_file: Path) -> Dict:
        """Compare Go and Python coverage."""
        go_stats = self.go_analyzer.get_coverage_stats(go_file) if go_file.exists() else None
        py_stats = self.py_analyzer.parse_coverage_json(py_file) if py_file.exists() else None

        comparison = {
            'go': go_stats,
            'python': py_stats,
            'difference': None,
        }

        if go_stats and py_stats:
            diff = go_stats.total_coverage - py_stats.total_coverage
            comparison['difference'] = {
                'absolute': round(diff, 2),
                'percentage': round((diff / py_stats.total_coverage * 100) if py_stats.total_coverage > 0 else 0, 2),
                'better_language': 'Go' if diff > 0 else 'Python',
            }

        return comparison

    def identify_gaps(self, go_file: Path, py_file: Path, threshold: float = 95.0) -> Dict:
        """Identify coverage gaps in both projects."""
        gaps = {'go': [], 'python': []}

        go_stats = self.go_analyzer.get_coverage_stats(go_file) if go_file.exists() else None
        if go_stats:
            for module, coverage in go_stats.modules.items():
                if coverage < threshold:
                    gaps['go'].append({
                        'module': module,
                        'coverage': round(coverage, 2),
                        'gap': round(threshold - coverage, 2),
                    })

        py_stats = self.py_analyzer.parse_coverage_json(py_file) if py_file.exists() else None
        if py_stats:
            for module, coverage in py_stats.modules.items():
                if coverage < threshold:
                    gaps['python'].append({
                        'module': module,
                        'coverage': round(coverage, 2),
                        'gap': round(threshold - coverage, 2),
                    })

        # Sort by gap
        gaps['go'].sort(key=lambda x: -x['gap'])
        gaps['python'].sort(key=lambda x: -x['gap'])

        return gaps

    def generate_text_report(self, go_file: Path, py_file: Path) -> str:
        """Generate text comparison report."""
        comparison = self.compare_coverage(go_file, py_file)
        gaps = self.identify_gaps(go_file, py_file)

        report = []
        report.append("=" * 80)
        report.append("COVERAGE COMPARISON REPORT")
        report.append("=" * 80)

        if comparison['go']:
            report.append("\n[GO PROJECT]")
            report.append(f"Total Coverage: {comparison['go'].total_coverage:.2f}%")
            report.append(f"Files: {comparison['go'].covered_files}/{comparison['go'].total_files}")
            report.append(f"Statements: {comparison['go'].covered_statements}/{comparison['go'].statements}")

        if comparison['python']:
            report.append("\n[PYTHON PROJECT]")
            report.append(f"Total Coverage: {comparison['python'].total_coverage:.2f}%")
            report.append(f"Files: {comparison['python'].covered_files}/{comparison['python'].total_files}")
            report.append(f"Statements: {comparison['python'].covered_statements}/{comparison['python'].statements}")

        if comparison['difference']:
            report.append(f"\n[DIFFERENCE]")
            report.append(f"Absolute: {comparison['difference']['absolute']:+.2f}%")
            report.append(f"Better: {comparison['difference']['better_language']}")

        # Coverage gaps
        report.append(f"\n{'=' * 80}")
        report.append("MODULES BELOW 95% COVERAGE")
        report.append("=" * 80)

        if gaps['go']:
            report.append("\nGo Modules:")
            for gap in gaps['go'][:10]:
                report.append(f"  {gap['module']}: {gap['coverage']:.2f}% (gap: {gap['gap']:.2f}%)")

        if gaps['python']:
            report.append("\nPython Modules:")
            for gap in gaps['python'][:10]:
                report.append(f"  {gap['module']}: {gap['coverage']:.2f}% (gap: {gap['gap']:.2f}%)")

        return "\n".join(report)

    def generate_html_report(self, go_file: Path, py_file: Path) -> str:
        """Generate HTML comparison report."""
        comparison = self.compare_coverage(go_file, py_file)
        gaps = self.identify_gaps(go_file, py_file)

        html = """<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Coverage Comparison Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        h1 { color: #333; }
        .section { background: white; padding: 20px; margin: 10px 0; border-radius: 5px; }
        .metric { display: inline-block; margin-right: 20px; }
        .metric-value { font-size: 28px; font-weight: bold; color: #2196F3; }
        .metric-label { color: #666; font-size: 12px; }
        table { border-collapse: collapse; width: 100%; margin-top: 10px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f5f5f5; }
        .good { color: green; }
        .bad { color: red; }
        .warning { color: orange; }
        .comparison-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
    </style>
</head>
<body>
    <h1>Coverage Comparison Report</h1>
"""

        # Coverage comparison
        html += '<div class="section">\n<h2>Coverage Metrics</h2>\n<div class="comparison-grid">\n'

        if comparison['go']:
            go_color = 'good' if comparison['go'].total_coverage >= 95 else 'warning' if comparison['go'].total_coverage >= 80 else 'bad'
            html += f"""
    <div>
        <h3>Go Project</h3>
        <div class="metric">
            <div class="metric-value {go_color}">{comparison['go'].total_coverage:.2f}%</div>
            <div class="metric-label">Total Coverage</div>
        </div>
        <table>
            <tr><td>Files</td><td>{comparison['go'].covered_files}/{comparison['go'].total_files}</td></tr>
            <tr><td>Statements</td><td>{comparison['go'].covered_statements}/{comparison['go'].statements}</td></tr>
        </table>
    </div>
"""

        if comparison['python']:
            py_color = 'good' if comparison['python'].total_coverage >= 95 else 'warning' if comparison['python'].total_coverage >= 80 else 'bad'
            html += f"""
    <div>
        <h3>Python Project</h3>
        <div class="metric">
            <div class="metric-value {py_color}">{comparison['python'].total_coverage:.2f}%</div>
            <div class="metric-label">Total Coverage</div>
        </div>
        <table>
            <tr><td>Files</td><td>{comparison['python'].covered_files}/{comparison['python'].total_files}</td></tr>
            <tr><td>Statements</td><td>{comparison['python'].covered_statements}/{comparison['python'].statements}</td></tr>
        </table>
    </div>
"""

        html += '</div>\n</div>\n'

        # Coverage gaps
        html += '<div class="section">\n<h2>Modules Below 95% Coverage</h2>\n'

        if gaps['go']:
            html += '<h3>Go Modules</h3>\n<table>\n<tr><th>Module</th><th>Coverage</th><th>Gap</th></tr>\n'
            for gap in gaps['go'][:15]:
                html += f"<tr><td>{gap['module']}</td><td class=\"bad\">{gap['coverage']:.2f}%</td><td>{gap['gap']:.2f}%</td></tr>\n"
            html += '</table>\n'

        if gaps['python']:
            html += '<h3>Python Modules</h3>\n<table>\n<tr><th>Module</th><th>Coverage</th><th>Gap</th></tr>\n'
            for gap in gaps['python'][:15]:
                html += f"<tr><td>{gap['module']}</td><td class=\"bad\">{gap['coverage']:.2f}%</td><td>{gap['gap']:.2f}%</td></tr>\n"
            html += '</table>\n'

        html += '</div>\n</body>\n</html>'

        return html


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Compare Go and Python coverage metrics",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python coverage_diff.py --go-coverage coverage.out --py-coverage coverage.json
  python coverage_diff.py --compare main --show-gaps
  python coverage_diff.py --report --output comparison.txt
  python coverage_diff.py --html report.html
        """
    )

    parser.add_argument(
        "--go-coverage", "-g",
        type=Path,
        default=Path("coverage.out"),
        help="Path to Go coverage.out file"
    )
    parser.add_argument(
        "--py-coverage", "-p",
        type=Path,
        default=Path("coverage.json"),
        help="Path to Python coverage.json file"
    )
    parser.add_argument(
        "--show-gaps",
        action="store_true",
        help="Show coverage gaps"
    )
    parser.add_argument(
        "--threshold",
        type=float,
        default=95.0,
        help="Coverage threshold for identifying gaps"
    )
    parser.add_argument(
        "--report", "-r",
        action="store_true",
        help="Generate text report"
    )
    parser.add_argument(
        "--html",
        type=Path,
        help="Generate HTML report to file"
    )
    parser.add_argument(
        "--output", "-o",
        type=Path,
        help="Output file for text report"
    )

    args = parser.parse_args()

    analyzer = CoverageDiffAnalyzer()

    if args.report or args.output:
        report = analyzer.generate_text_report(args.go_coverage, args.py_coverage)
        if args.output:
            args.output.write_text(report)
            print(f"Report written to: {args.output}")
        else:
            print(report)
        return 0

    if args.html:
        report = analyzer.generate_html_report(args.go_coverage, args.py_coverage)
        args.html.write_text(report)
        print(f"HTML report written to: {args.html}")
        return 0

    if args.show_gaps:
        gaps = analyzer.identify_gaps(args.go_coverage, args.py_coverage, args.threshold)
        print(f"Modules below {args.threshold}% coverage:")
        if gaps['go']:
            print("  Go:", gaps['go'])
        if gaps['python']:
            print("  Python:", gaps['python'])
        return 0

    # Default: show comparison
    comparison = analyzer.compare_coverage(args.go_coverage, args.py_coverage)
    if comparison['go']:
        print(f"Go Coverage: {comparison['go'].total_coverage:.2f}%")
    if comparison['python']:
        print(f"Python Coverage: {comparison['python'].total_coverage:.2f}%")
    if comparison['difference']:
        print(f"Difference: {comparison['difference']['absolute']:+.2f}%")

    return 0


if __name__ == "__main__":
    sys.exit(main())
