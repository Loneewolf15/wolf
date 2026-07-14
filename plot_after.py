import matplotlib.pyplot as plt
import numpy as np

languages = ['Wolf\n(Optimized)', 'Rust\n(Tokio)', 'Go\n(net/http)']
rps = [5035, 4646, 4347]
avg_latency_ms = [27.5, 16.3, 21.5]

fig, ax1 = plt.subplots(figsize=(10, 6))

color = 'tab:blue'
ax1.set_xlabel('Runtime / Language', fontsize=12)
ax1.set_ylabel('Requests per Second (RPS) - Higher is Better', color=color, fontsize=12)
bars = ax1.bar(languages, rps, color=color, alpha=0.7, width=0.4)
ax1.tick_params(axis='y', labelcolor=color)
ax1.set_ylim(0, 6000)

for bar in bars:
    yval = bar.get_height()
    ax1.text(bar.get_x() + bar.get_width()/2, yval + 50, f'{yval:,}', ha='center', va='bottom', fontweight='bold', color=color)

ax2 = ax1.twinx()  
color = 'tab:red'
ax2.set_ylabel('Avg Latency (ms) - Lower is Better', color=color, fontsize=12)  
lines = ax2.plot(languages, avg_latency_ms, color=color, marker='o', linewidth=2, markersize=8)
ax2.tick_params(axis='y', labelcolor=color)
ax2.set_ylim(0, 40)

for i, txt in enumerate(avg_latency_ms):
    ax2.annotate(f'{txt}ms', (languages[i], avg_latency_ms[i]), 
                 textcoords="offset points", xytext=(0, -20) if i == 0 else (0, 10), ha='center', color=color, fontweight='bold')

plt.title('Real-World Workload Benchmark: Wolf vs Systems Languages', fontsize=14, fontweight='bold')
fig.tight_layout()  

plt.savefig('benchmark_results_after.png', dpi=300)
