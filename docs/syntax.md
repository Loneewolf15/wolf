# Wolf Syntax Reference 🐺

Wolf syntax is designed to be expressive, safe, and familiar. Variables use the `$` prefix, and control flow is C-style but without parentheses. 

## Variables & Types

All variables must start with `$`. By default, variables are dynamically typed (`interface{}` under the hood in Go), but you can use `var` for strict typing.

```wolf
# Dynamic assignment (automatically uses := the first time, = thereafter)
$name = "Ghost"
$age = 5
$is_direwolf = true
$score = 99.5

# Strict typing
var $id: int = 12345
var $token: string = "abc"
```

## String Interpolation

Use `{ }` braces inside double-quoted strings for interpolation:

```wolf
$first = "Jon"
$last = "Snow"
print("You know nothing, {$first} {$last}.")
```

## Data Structures

Wolf uses Python-like syntax for slices and maps.

```wolf
# Array / Slice
$pack = ["Ghost", "Nymeria", "Summer", "Shaggydog", "Grey Wind", "Lady"]

# Map / Dictionary
$stats = {
    "health": 100,
    "stamina": 85.5
}
```

## Control Flow

### If / Else
No parentheses required around conditions.

```wolf
if $age < 1 {
    print("Pup")
} else if $age < 3 {
    print("Young wolf")
} else {
    print("Adult wolf")
}
```

### Match (Switch)
Wolf uses `match` for pattern matching, utilizing `=>` for arms.

```wolf
match $state {
    "idle" => {
        print("Resting")
    }
    "hunting" => {
        print("On the prowl")
    }
    _ => {
        print("Unknown state")
    }
}
```

### For Loops
C-style for loops and foreach loops are supported.

```wolf
# Standard for loop
for $i = 0; $i < 10; $i++ {
    print($i)
}

# Range / Foreach loop over arrays
foreach $pack as $wolf {
    print("Wolf: {$wolf}")
}

# Range / Foreach loop over maps
foreach $stats as $key => $val {
    print("{$key}: {$val}")
}
```

## Functions

Functions can be dynamically typed or statically typed. Untyped parameters in typed functions infer their types from the return type.

```wolf
# Dynamic function
func greet($name) {
    print("Hello {$name}")
}

# Statically typed function
func add($a, $b) -> float {
    return $a + $b
}
```

## Classes

Wolf supports Object-Oriented patterns via `class`, which compiles down to Go structs and receiver methods.

```wolf
class Wolf {
    $name: string
    $age: int

    # Constructor
    func __construct($name, $age) {
        $this->name = $name
        $this->age = $age
    }

    func howl() {
        print("{$this->name} howls at the moon!")
    }
}

$ghost = new Wolf("Ghost", 5)
$ghost->howl()
```
