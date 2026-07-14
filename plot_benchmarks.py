import matplotlib.pyplot as plt
import numpy as np

languages = ['Rust', 'Wolf', 'Go', 'Node.js', 'C', 'Python']
rps = [4967, 4870, 4352, 2838, 1285, 778]
avg_latency_ms = [15.3, 28.3, 21.6, 46.6, 70.9, 184.0]

fig, ax1 = plt.subplots(figsize=(10, 6))

# Plot RPS as bar chart
color = 'tab:blue'
ax1.set_xlabel('Language', fontsize=12)
ax1.set_ylabel('Requests per Second (RPS)', color=color, fontsize=12)
bars = ax1.bar(languages, rps, color=color, alpha=0.7)
ax1.tick_params(axis='y', labelcolor=color)

# Add RPS labels
for bar in bars:
    yval = bar.get_height()
    ax1.text(bar.get_x() + bar.get_width()/2, yval + 50, f'{yval:,}', ha='center', va='bottom', fontweight='bold')

# Create a secondary y-axis for latency
ax2 = ax1.twinx()  
color = 'tab:red'
ax2.set_ylabel('Avg Latency (ms) - Lower is Better', color=color, fontsize=12)  
lines = ax2.plot(languages, avg_latency_ms, color=color, marker='o', linewidth=2, markersize=8)
ax2.tick_params(axis='y', labelcolor=color)
ax2.set_ylim(0, max(avg_latency_ms) * 1.2)

# Add Latency labels
for i, txt in enumerate(avg_latency_ms):
    ax2.annotate(f'{txt}ms', (languages[i], avg_latency_ms[i]), 
                 textcoords="offset points", xytext=(0,10), ha='center', color=color, fontweight='bold')

plt.title('Benchmark Results: RPS vs Avg Latency (150 concurrent workers)', fontsize=14, fontweight='bold')
fig.tight_layout()  

plt.savefig('benchmark_results.png', dpi=300)
print("Graph saved to benchmark_results.png")
