#!/usr/bin/env python3
"""
Benchmark Runner for ZK Health Infrastructure

This script runs all benchmarks and aggregates the results
"""

import sys
import json
import time
from rich.console import Console
from rich.table import Table

# Import benchmark modules
from benchmark_identity import run_identity_benchmarks
from benchmark_document import run_document_benchmarks
from benchmark_policy import run_policy_benchmarks
from benchmark_gateway import run_gateway_benchmarks
from benchmark_infrastructure import run_infrastructure_benchmarks

console = Console()

def main():
    """Run all benchmarks and display results"""
    console.print("\n[bold blue]======= ZK Health Infrastructure Benchmarks =======[/bold blue]\n")
    
    # Track start time for overall benchmark duration
    start_time = time.time()
    
    # Run all benchmarks
    console.print("[bold]Running Identity Management Benchmarks...[/bold]")
    identity_results = run_identity_benchmarks(100)
    console.print("\n[green]✓[/green] Identity Management Benchmarks completed\n")
    
    console.print("[bold]Running Document Management Benchmarks...[/bold]")
    document_results = run_document_benchmarks(100)
    console.print("\n[green]✓[/green] Document Management Benchmarks completed\n")
    
    console.print("[bold]Running Policy Validation Benchmarks...[/bold]")
    policy_results = run_policy_benchmarks(100)
    console.print("\n[green]✓[/green] Policy Validation Benchmarks completed\n")
    
    console.print("[bold]Running API Gateway Benchmarks...[/bold]")
    gateway_results = run_gateway_benchmarks(100)
    console.print("\n[green]✓[/green] API Gateway Benchmarks completed\n")
    
    console.print("[bold]Running Infrastructure Benchmarks...[/bold]")
    infrastructure_results = run_infrastructure_benchmarks(100)
    console.print("\n[green]✓[/green] Infrastructure Benchmarks completed\n")
    
    # Calculate total time
    total_time = time.time() - start_time
    
    # Display summary table
    console.print("[bold]Benchmark Summary:[/bold]")
    
    table = Table(show_header=True, header_style="bold")
    table.add_column("Category")
    table.add_column("Operation")
    table.add_column("Avg Time (ms)")
    table.add_column("Throughput (ops/sec)")
    
    # Add identity results
    for op, metrics in identity_results.items():
        table.add_row("Identity", op.replace("_", " ").title(), 
                     f"{metrics['avg_time']:.2f}", 
                     f"{metrics['throughput']:.2f}")
    
    # Add document results
    for op, metrics in document_results.items():
        table.add_row("Document", op.replace("_", " ").title(), 
                     f"{metrics['avg_time']:.2f}", 
                     f"{metrics['throughput']:.2f}")
    
    # Add policy results
    for op, metrics in policy_results.items():
        table.add_row("Policy", op.replace("_", " ").title(), 
                     f"{metrics['avg_time']:.2f}", 
                     f"{metrics['throughput']:.2f}")
    
    # Add gateway results
    for op, metrics in gateway_results.items():
        table.add_row("Gateway", op.replace("_", " ").title(), 
                     f"{metrics['avg_time']:.2f}", 
                     f"{metrics['throughput']:.2f}")
    
    # Add infrastructure results
    for op, metrics in infrastructure_results.items():
        if isinstance(metrics, dict) and "avg_time" in metrics:
            table.add_row("Infrastructure", op.replace("_", " ").title(), 
                         f"{metrics['avg_time']:.2f}",
                         f"{metrics.get('throughput', 'N/A')}")
    
    console.print(table)
    
    # Display total time
    console.print(f"\n[bold]Total benchmark time:[/bold] {total_time:.2f} seconds")
    
    # Save results to JSON
    all_results = {
        "identity": identity_results,
        "document": document_results,
        "policy": policy_results,
        "gateway": gateway_results,
        "infrastructure": infrastructure_results,
        "total_time": total_time
    }
    
    with open("benchmark_results.json", "w") as f:
        json.dump(all_results, f, indent=2)
    
    console.print("\n[bold green]Benchmark results saved to benchmark_results.json[/bold green]")
    
    # Update logs
    update_logs(all_results)
    console.print("[bold green]Logs updated with benchmark results[/bold green]")
    
    return 0

def update_logs(results):
    """Update the logs.md file with benchmark results"""
    timestamp = time.strftime("%Y-%m-%d %H:%M:%S")
    
    log_entry = f"""
## Benchmark Results - {timestamp}

### Identity Management
"""
    
    for op, metrics in results["identity"].items():
        log_entry += f"- **{op.replace('_', ' ').title()}**: {metrics['avg_time']:.2f}ms, {metrics['throughput']:.2f} ops/sec\n"
    
    log_entry += "\n### Document Management\n"
    for op, metrics in results["document"].items():
        log_entry += f"- **{op.replace('_', ' ').title()}**: {metrics['avg_time']:.2f}ms, {metrics['throughput']:.2f} ops/sec\n"
    
    log_entry += "\n### Policy Validation\n"
    for op, metrics in results["policy"].items():
        log_entry += f"- **{op.replace('_', ' ').title()}**: {metrics['avg_time']:.2f}ms, {metrics['throughput']:.2f} ops/sec\n"
    
    log_entry += "\n### API Gateway\n"
    for op, metrics in results["gateway"].items():
        log_entry += f"- **{op.replace('_', ' ').title()}**: {metrics['avg_time']:.2f}ms, {metrics['throughput']:.2f} ops/sec\n"
    
    log_entry += "\n### Infrastructure\n"
    for op, metrics in results["infrastructure"].items():
        if isinstance(metrics, dict) and "avg_time" in metrics:
            throughput = metrics.get('throughput', 'N/A')
            throughput_str = f"{throughput:.2f} ops/sec" if isinstance(throughput, (int, float)) else throughput
            log_entry += f"- **{op.replace('_', ' ').title()}**: {metrics['avg_time']:.2f}ms, {throughput_str}\n"
    
    log_entry += f"\n**Total benchmark time**: {results['total_time']:.2f} seconds\n"
    
    # Append to logs.md
    with open("logs.md", "a") as f:
        f.write(log_entry)

if __name__ == "__main__":
    sys.exit(main())
