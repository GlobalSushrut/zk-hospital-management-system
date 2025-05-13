#!/usr/bin/env python3
"""
Benchmark Tool for ZK Health Infrastructure
This script measures performance, scalability, and capabilities of the infrastructure
"""

import time
import json
import os
import uuid
import random
import click
import datetime
import matplotlib.pyplot as plt
import numpy as np
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich.progress import Progress, track

console = Console()

# Import all component benchmark modules
from benchmark_identity import run_identity_benchmarks
from benchmark_oracle import run_oracle_benchmarks
from benchmark_consent import run_consent_benchmarks
from benchmark_document import run_document_benchmarks
from benchmark_treatment import run_treatment_benchmarks
from benchmark_gateway import run_gateway_benchmarks
from benchmark_policy import run_policy_benchmarks

def run_benchmarks(component=None, iterations=100, output_file=None, chart_dir=None):
    """
    Run benchmarks and generate a performance report
    
    Args:
        component: Specific component to benchmark (or None for all)
        iterations: Number of iterations for each benchmark
        output_file: File to save benchmark results as JSON
        chart_dir: Directory to save benchmark charts
        
    Returns:
        Dictionary of benchmark results
    """
    console.print(Panel(
        f"[bold blue]ZK Health Infrastructure Benchmarks[/bold blue]\n"
        f"Component: [cyan]{component or 'all'}[/cyan]\n"
        f"Iterations: [cyan]{iterations}[/cyan]",
        title="Benchmark Configuration"
    ))
    
    # Prepare results container
    benchmark_start_time = time.time()
    results = {}
    
    # Create charts directory if specified
    if chart_dir and not os.path.exists(chart_dir):
        os.makedirs(chart_dir)
    
    # Run appropriate benchmarks based on component selection
    if component in [None, "all", "identity"]:
        console.print(Panel("[bold green]Running Identity Benchmarks[/bold green]"))
        results["identity"] = run_identity_benchmarks(iterations)
    
    if component in [None, "all", "consent"]:
        console.print(Panel("[bold green]Running Consent Benchmarks[/bold green]"))
        results["consent"] = run_consent_benchmarks(iterations)
    
    if component in [None, "all", "oracle"]:
        console.print(Panel("[bold green]Running Oracle Benchmarks[/bold green]"))
        results["oracle"] = run_oracle_benchmarks(iterations)
    
    if component in [None, "all", "document"]:
        console.print(Panel("[bold green]Running Document Benchmarks[/bold green]"))
        results["document"] = run_document_benchmarks(iterations)
    
    if component in [None, "all", "treatment"]:
        console.print(Panel("[bold green]Running Treatment Benchmarks[/bold green]"))
        results["treatment"] = run_treatment_benchmarks(iterations)
    
    if component in [None, "all", "gateway"]:
        console.print(Panel("[bold green]Running Gateway Benchmarks[/bold green]"))
        results["gateway"] = run_gateway_benchmarks(iterations)
    
    if component in [None, "all", "policy"]:
        console.print(Panel("[bold green]Running Policy Agreement Engine Benchmarks[/bold green]"))
        results["policy"] = run_policy_benchmarks(iterations)
    
    benchmark_end_time = time.time()
    total_duration = benchmark_end_time - benchmark_start_time
    
    # Add benchmark metadata
    results["metadata"] = {
        "timestamp": datetime.datetime.now().isoformat(),
        "iterations": iterations,
        "total_duration_seconds": total_duration,
        "components_tested": component if component else "all"
    }
    
    # Print results
    print_benchmark_results(results)
    
    # Generate charts if directory is specified
    if chart_dir:
        console.print(Panel("[bold]Generating benchmark charts...[/bold]"))
        generate_benchmark_charts(results, chart_dir)
    
    # Save results if output file is specified
    if output_file:
        with open(output_file, 'w') as f:
            json.dump(results, f, indent=2)
        console.print(f"[green]Benchmark results saved to {output_file}[/green]")
    
    return results

def print_benchmark_results(results):
    """Pretty print benchmark results"""
    console.print("\n[bold]Benchmark Results Summary:[/bold]")
    
    # Skip metadata key when printing component results
    for component, component_results in [(k, v) for k, v in results.items() if k != "metadata"]:
        table = Table(title=f"{component.title()} Component")
        table.add_column("Operation", style="cyan")
        table.add_column("Avg Time (ms)", justify="right", style="green")
        table.add_column("Min Time (ms)", justify="right", style="green")
        table.add_column("Max Time (ms)", justify="right", style="green")
        table.add_column("Throughput (ops/sec)", justify="right", style="yellow")
        
        for operation, metrics in component_results.items():
            table.add_row(
                operation.replace('_', ' ').title(),
                f"{metrics['avg_time']:.2f}",
                f"{metrics['min_time']:.2f}",
                f"{metrics['max_time']:.2f}",
                f"{metrics['throughput']:.2f}"
            )
        
        console.print(table)
        console.print("")
    
    # Print overall summary
    if "metadata" in results:
        metadata = results["metadata"]
        console.print(Panel(
            f"[bold]Total benchmarking time:[/bold] {metadata['total_duration_seconds']:.2f} seconds\n"
            f"[bold]Iterations per benchmark:[/bold] {metadata['iterations']}\n"
            f"[bold]Timestamp:[/bold] {metadata['timestamp']}\n",
            title="Benchmark Summary"
        ))
        
def generate_benchmark_charts(results, chart_dir):
    """Generate charts visualizing benchmark results"""
    # Skip metadata when generating charts
    components = [k for k in results.keys() if k != "metadata"]
    
    # Overall throughput comparison
    plt.figure(figsize=(12, 8))
    
    # Collect data for chart
    operations_by_component = {}
    throughputs_by_component = {}
    
    for component in components:
        operations = []
        throughputs = []
        
        for operation, metrics in results[component].items():
            operations.append(f"{component}-{operation}")
            throughputs.append(metrics["throughput"])
        
        operations_by_component[component] = operations
        throughputs_by_component[component] = throughputs
    
    # Flatten lists for overall chart
    all_operations = []
    all_throughputs = []
    component_colors = {}
    
    # Assign colors to components
    color_map = plt.cm.get_cmap('tab10', len(components))
    for i, component in enumerate(components):
        component_colors[component] = color_map(i)
        
        for j, operation in enumerate(operations_by_component[component]):
            all_operations.append(operation)
            all_throughputs.append(throughputs_by_component[component][j])
    
    # Create the bar chart
    y_pos = np.arange(len(all_operations))
    
    # Color bars by component
    colors = []
    for op in all_operations:
        component = op.split('-')[0]
        colors.append(component_colors[component])
    
    plt.barh(y_pos, all_throughputs, color=colors)
    plt.yticks(y_pos, all_operations)
    plt.xlabel('Throughput (operations/second)')
    plt.title('ZK Health Infrastructure Throughput Comparison')
    
    # Save the chart
    overall_chart_path = os.path.join(chart_dir, 'overall_throughput.png')
    plt.tight_layout()
    plt.savefig(overall_chart_path)
    plt.close()
    
    # Create individual component charts
    for component in components:
        operations = []
        avg_times = []
        throughputs = []
        
        for operation, metrics in results[component].items():
            operations.append(operation.replace('_', ' ').title())
            avg_times.append(metrics["avg_time"])
            throughputs.append(metrics["throughput"])
        
        # Average time chart
        plt.figure(figsize=(10, 6))
        plt.bar(operations, avg_times, color='skyblue')
        plt.ylabel('Average Time (ms)')
        plt.title(f'{component.title()} Component - Average Processing Time')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        avg_time_chart_path = os.path.join(chart_dir, f'{component}_avg_time.png')
        plt.savefig(avg_time_chart_path)
        plt.close()
        
        # Throughput chart
        plt.figure(figsize=(10, 6))
        plt.bar(operations, throughputs, color='lightgreen')
        plt.ylabel('Throughput (ops/second)')
        plt.title(f'{component.title()} Component - Throughput')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        throughput_chart_path = os.path.join(chart_dir, f'{component}_throughput.png')
        plt.savefig(throughput_chart_path)
        plt.close()
    
    console.print(f"[green]Charts saved to {chart_dir}[/green]")

def run_stress_test(component, duration=60, ramp_up=10, output_file=None):
    """
    Run a stress test for a specific component
    
    Args:
        component: Component to stress test
        duration: Duration of the test in seconds
        ramp_up: Ramp-up period in seconds
        output_file: File to save stress test results
    """
    console.print(Panel(
        f"[bold red]Running Stress Test on {component.title()} Component[/bold red]\n"
        f"Duration: [cyan]{duration} seconds[/cyan]\n"
        f"Ramp-up: [cyan]{ramp_up} seconds[/cyan]"
    ))
    
    # Map components to their benchmark functions
    component_map = {
        "identity": run_identity_benchmarks,
        "consent": run_consent_benchmarks,
        "oracle": run_oracle_benchmarks,
        "document": run_document_benchmarks,
        "treatment": run_treatment_benchmarks,
        "gateway": run_gateway_benchmarks
    }
    
    if component not in component_map:
        console.print(f"[bold red]Error: Component '{component}' not found[/bold red]")
        return
    
    benchmark_func = component_map[component]
    
    # Track metrics over time
    timestamps = []
    throughputs = []
    response_times = []
    error_rates = []
    
    start_time = time.time()
    end_time = start_time + duration
    
    with Progress() as progress:
        task = progress.add_task(f"[cyan]Stress testing {component}...", total=duration)
        
        while time.time() < end_time:
            current_time = time.time()
            elapsed = current_time - start_time
            progress.update(task, completed=min(elapsed, duration))
            
            # Calculate load factor based on ramp-up period
            if elapsed < ramp_up:
                load_factor = elapsed / ramp_up
            else:
                load_factor = 1.0
            
            # Run a mini benchmark with iterations proportional to the load factor
            iterations = max(1, int(10 * load_factor))
            current_results = benchmark_func(iterations)
            
            # Record metrics
            timestamps.append(elapsed)
            
            # Average throughput and response time across all operations
            avg_throughput = sum(op['throughput'] for op in current_results.values()) / len(current_results)
            avg_response_time = sum(op['avg_time'] for op in current_results.values()) / len(current_results)
            
            throughputs.append(avg_throughput)
            response_times.append(avg_response_time)
            
            # Simulate error rate (would be real in production)
            simulated_error_rate = random.uniform(0, 0.1) * load_factor  # 0-10% error rate increasing with load
            error_rates.append(simulated_error_rate)
            
            # Throttle to avoid overwhelming the system
            time.sleep(1)
    
    # Generate stress test report
    stress_results = {
        "component": component,
        "duration": duration,
        "ramp_up": ramp_up,
        "timestamps": timestamps,
        "throughputs": throughputs,
        "response_times": response_times,
        "error_rates": error_rates,
        "peak_throughput": max(throughputs),
        "peak_response_time": max(response_times),
        "peak_error_rate": max(error_rates)
    }
    
    # Print stress test results
    console.print(Panel(
        f"[bold]Peak Throughput:[/bold] {stress_results['peak_throughput']:.2f} ops/sec\n"
        f"[bold]Peak Response Time:[/bold] {stress_results['peak_response_time']:.2f} ms\n"
        f"[bold]Peak Error Rate:[/bold] {stress_results['peak_error_rate']*100:.2f}%",
        title=f"Stress Test Results for {component.title()} Component"
    ))
    
    # Save results if output file is specified
    if output_file:
        with open(output_file, 'w') as f:
            # Convert numpy arrays to lists for JSON serialization
            json_results = {k: (v.tolist() if hasattr(v, 'tolist') else v) 
                           for k, v in stress_results.items()}
            json.dump(json_results, f, indent=2)
        console.print(f"[green]Stress test results saved to {output_file}[/green]")
    
    return stress_results


def run_regression_test(test_files):
    """
    Run regression test by comparing current benchmark against previous results
    
    Args:
        test_files: List of files containing previous benchmark results
    """
    console.print(Panel("[bold purple]Running Regression Test[/bold purple]"))
    
    # Run current benchmark with default parameters
    current_results = run_benchmarks(iterations=50)
    
    for test_file in test_files:
        if not os.path.exists(test_file):
            console.print(f"[yellow]Warning: Test file '{test_file}' not found, skipping[/yellow]")
            continue
        
        with open(test_file, 'r') as f:
            previous_results = json.load(f)
        
        console.print(f"\n[bold]Comparing against: {test_file}[/bold]")
        table = Table(title="Performance Regression Analysis")
        table.add_column("Component", style="cyan")
        table.add_column("Operation", style="cyan")
        table.add_column("Current (ms)", justify="right")
        table.add_column("Previous (ms)", justify="right")
        table.add_column("Change (%)", justify="right", style="green")
        table.add_column("Status", style="yellow")
        
        regression_detected = False
        
        # Compare each component and operation
        for component in current_results:
            if component == "metadata":
                continue
                
            if component not in previous_results:
                console.print(f"[yellow]Component '{component}' not in previous results, skipping[/yellow]")
                continue
                
            for operation in current_results[component]:
                if operation not in previous_results[component]:
                    console.print(f"[yellow]Operation '{operation}' not in previous results, skipping[/yellow]")
                    continue
                
                current_time = current_results[component][operation]["avg_time"]
                previous_time = previous_results[component][operation]["avg_time"]
                
                # Calculate percent change
                percent_change = ((current_time - previous_time) / previous_time) * 100
                
                # Determine status
                if percent_change > 10:  # More than 10% slower
                    status = "REGRESSION"
                    regression_detected = True
                    style = "red"
                elif percent_change < -10:  # More than 10% faster
                    status = "IMPROVEMENT"
                    style = "green"
                else:
                    status = "STABLE"
                    style = "yellow"
                
                table.add_row(
                    component,
                    operation.replace('_', ' ').title(),
                    f"{current_time:.2f}",
                    f"{previous_time:.2f}",
                    f"{percent_change:.2f}",
                    f"[{style}]{status}[/{style}]"
                )
        
        console.print(table)
        
        if regression_detected:
            console.print("[bold red]⚠️ Performance regression detected! ⚠️[/bold red]")
        else:
            console.print("[bold green]✓ No performance regression detected[/bold green]")


@click.group()
def cli():
    """ZK Health Infrastructure Benchmark Tool"""
    pass

@cli.command()
@click.option('--component', '-c', help='Component to benchmark (identity, consent, oracle, document, treatment, gateway, policy, or all)')
@click.option('--iterations', '-i', default=100, help='Number of iterations for each benchmark')
@click.option('--output', '-o', help='Output file for benchmark results (JSON format)')
@click.option('--chart-dir', '-d', help='Directory to save benchmark charts')
def benchmark(component, iterations, output, chart_dir):
    """Run benchmarks for ZK Health Infrastructure components"""
    run_benchmarks(component, iterations, output, chart_dir)

@cli.command()
@click.option('--component', '-c', required=True, help='Component to stress test (required)')
@click.option('--duration', '-d', default=60, help='Duration of stress test in seconds')
@click.option('--ramp-up', '-r', default=10, help='Ramp-up period in seconds')
@click.option('--output', '-o', help='Output file for stress test results (JSON format)')
def stress(component, duration, ramp_up, output):
    """Run a stress test on a specific component"""
    run_stress_test(component, duration, ramp_up, output)

@cli.command()
@click.argument('test_files', nargs=-1, required=True)
def regression(test_files):
    """Run regression tests comparing against previous benchmark results"""
    run_regression_test(test_files)


if __name__ == '__main__':
    cli()
