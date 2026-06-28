import os
import re
import matplotlib.pyplot as plt

files = {
    'Go': 'results_real_go.txt',
    'Node': 'results_real_node.txt',
    'Python': 'results_real_python.txt',
    'Wolf': 'results_real_wolf.txt',
    'Rust': 'results_real_rust.txt',
    'C': 'results_real_c.txt'
}

rps_data = {}
p95_data = {}

for lang, filename in files.items():
    filepath = os.path.join('bench', filename)
    if not os.path.exists(filepath):
        print(f"Warning: {filepath} not found.")
        continue
    
    with open(filepath, 'r') as f:
        content = f.read()
        
        rps_match = re.search(r'Req/s:\s+([\d.]+)', content)
        p95_match = re.search(r'Avg Latency:\s+([\d.]+)ms', content)
        
        if rps_match and p95_match:
            rps_data[lang] = float(rps_match.group(1))
            p95_data[lang] = float(p95_match.group(1))
        else:
            print(f"Warning: Could not parse metrics in {filepath}")

if not rps_data:
    print("No data found to plot.")
    exit(1)

# Sort by RPS (highest first)
sorted_langs = sorted(rps_data.keys(), key=lambda x: rps_data[x], reverse=True)
rps_values = [rps_data[lang] for lang in sorted_langs]
p95_values = [p95_data[lang] for lang in sorted_langs]

fig, ax1 = plt.subplots(figsize=(10, 6))

color1 = 'tab:blue'
ax1.set_xlabel('Language / Runtime')
ax1.set_ylabel('Requests per Second (RPS)', color=color1)
bars = ax1.bar(sorted_langs, rps_values, color=color1, alpha=0.7, label='RPS')
ax1.tick_params(axis='y', labelcolor=color1)

# Add values on top of bars
for bar in bars:
    yval = bar.get_height()
    ax1.text(bar.get_x() + bar.get_width()/2, yval, f'{int(yval)}', ha='center', va='bottom')

ax2 = ax1.twinx()
color2 = 'tab:red'
ax2.set_ylabel('Avg Latency (ms)', color=color2)
line = ax2.plot(sorted_langs, p95_values, color=color2, marker='o', linewidth=2, label='Avg Latency')
ax2.tick_params(axis='y', labelcolor=color2)

# Add values to line points
for i, v in enumerate(p95_values):
    ax2.text(i, v + (max(p95_values)*0.05), f'{v:.1f}ms', ha='center', va='bottom', color=color2, fontweight='bold')

plt.title('Wolf Real-World Load Benchmark vs Go, Node, Python, Rust, and C')
fig.tight_layout()

plt.savefig('bench/results_graph.png', dpi=300)
print("Graph generated: bench/results_graph.png")
