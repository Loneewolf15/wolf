# Machine Learning Bridge (`@ml`) 🐺

Wolf's defining feature is its seamless, native integration with Python and the vast ML ecosystem. Instead of writing clunky FFI (Foreign Function Interface) wrappers, writing boilerplate microservices, or dealing with fragile sub-processes, Wolf allows you to write Python directly inside your Go-compiled Wolf source code using the `@ml` block.

## How It Works

When the Wolf compiler encounters an `@ml` block, it extracts the Python code, sets up a persistent background Python worker process, and establishes a blazing-fast JSON-RPC bridge over streams. 

Variables specified in the `in` block are automatically serialized into the Python environment. Variables in the `out` block are automatically extracted from Python and deserialized back into Wolf's memory space. 

This bridge is entirely invisible to the user.

## The `@ml` Block

```wolf
$data = "Hello from Wolf"

@ml in: [$data] out: [$processed] {
    # This block executes natively in Python
    processed = data.upper() + " AND PYTHON"
}

print("Result: {$processed}")
```

### Syntax Overview
- `@ml`: Denotes the start of an ML block.
- `in: [$var1, $var2]`: Declares which Wolf variables should be injected into Python.
- `out: [$var_out]`: Declares which Python variables should be extracted back to Wolf.
- `{ ... }`: The Python source code.

## Automatic Environment Management

Wolf manages your Python dependencies automatically via the `wolf.python` file, created when you scaffold a project with `wolf new`.

**`wolf.python`**
```json
{
  "python_version": "3",
  "packages": [
    "numpy",
    "pandas",
    "scikit-learn"
  ],
  "venv_path": ".wolf-venv",
  "auto_activate": true
}
```

The first time you run your Wolf app, the compiler will bootstrap the virtual environment, install the missing dependencies via pip, and start the bridge.

## AI / ML Workloads

Using heavy ML libraries in Go natively is historically difficult. With Wolf, it takes 5 lines of code:

```wolf
# Load data array
$matrix = [1, 2, 3, 4, 5]

@ml in: [$matrix] out: [$mean, $std_dev] {
    import numpy as np
    
    arr = np.array(matrix)
    mean = float(np.mean(arr))
    std_dev = float(np.std(arr))
}

print("Mean: {$mean}")
print("Std Dev: {$std_dev}")
```

## Advanced Features

### Async ML Blocks
Machine learning inferences can take time. Wolf provides an `async`/`await` pattern that runs the Python block in a separate Go routine.

```wolf
$image_path = "/path/to/img.jpg"

# The async block returns a task channel immediately
$task = async @ml in: [$image_path] out: [$prediction] {
    import torch
    # ... load model and predict taking 5 seconds ...
    prediction = "cat"
}

print("Waiting for ML inference...")

# We block until the python bridge resolves
$result = await $task

print("Prediction: {$result}")
```

### Context Persistence
The Python background worker is persistent. If you import a massive 4GB LLM in one `@ml` block, it stays in memory. Subsequent `@ml` blocks can reuse the initial memory footprint without reloading the weights.
