; ModuleID = 'main'
source_filename = "main"
target datalayout = "e-m:e-p270:32:32-p271:32:32-p272:64:64-i64:64-f80:128-n8:16:32:64-S128"

define ptr @assert(ptr %cond_param, ptr %msg_param) {
entry:
  %cond = alloca ptr
  store ptr %cond_param, ptr %cond
  %msg = alloca ptr
  store ptr %msg_param, ptr %msg
  %0 = load ptr, ptr %cond
  %1 = call i1 @wolf_val_truthy(ptr %0)
  %2 = xor i1 %1, 1
  %3 = zext i1 %2 to i32
  %4 = call ptr @wolf_val_bool(i32 %3)
  %5 = call i1 @wolf_val_truthy(ptr %4)
  br i1 %5, label %if.then0, label %if.end1
if.then0:
  %6 = load ptr, ptr %msg
  %7 = call ptr @wolf_error(ptr %6)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @Counter_new() {
entry:
  %this = alloca ptr
  %0 = call ptr @wolf_class_create(ptr null)
  store ptr %0, ptr %this
  %1 = call ptr @wolf_val_int(i64 0)
  %2 = load ptr, ptr %this
  %3 = bitcast ptr @.str.1 to ptr
  call void @wolf_map_set(ptr %2, ptr %3, ptr %1)
  %4 = load ptr, ptr %this
  ret ptr %4
}

define ptr @Counter_increment(ptr %this_param) {
entry:
  %this = alloca ptr
  store ptr %this_param, ptr %this
  %0 = load ptr, ptr %this
  %1 = bitcast ptr @.str.1 to ptr
  %2 = call ptr @wolf_map_get(ptr %0, ptr %1)
  %3 = call ptr @wolf_val_int(i64 1)
  %4 = call i64 @wolf_val_to_i64(ptr %2)
  %5 = call i64 @wolf_val_to_i64(ptr %3)
  %6 = add i64 %4, %5
  %7 = call ptr @wolf_val_int(i64 %6)
  %8 = load ptr, ptr %this
  %9 = bitcast ptr @.str.1 to ptr
  call void @wolf_map_set(ptr %8, ptr %9, ptr %7)
  ret ptr null
}

define ptr @Counter_decrement(ptr %this_param) {
entry:
  %this = alloca ptr
  store ptr %this_param, ptr %this
  %0 = load ptr, ptr %this
  %1 = bitcast ptr @.str.1 to ptr
  %2 = call ptr @wolf_map_get(ptr %0, ptr %1)
  %3 = call ptr @wolf_val_int(i64 1)
  %4 = call i64 @wolf_val_to_i64(ptr %2)
  %5 = call i64 @wolf_val_to_i64(ptr %3)
  %6 = sub i64 %4, %5
  %7 = call ptr @wolf_val_int(i64 %6)
  %8 = load ptr, ptr %this
  %9 = bitcast ptr @.str.1 to ptr
  call void @wolf_map_set(ptr %8, ptr %9, ptr %7)
  ret ptr null
}

define ptr @Counter_reset(ptr %this_param) {
entry:
  %this = alloca ptr
  store ptr %this_param, ptr %this
  %0 = call ptr @wolf_val_int(i64 0)
  %1 = load ptr, ptr %this
  %2 = bitcast ptr @.str.1 to ptr
  call void @wolf_map_set(ptr %1, ptr %2, ptr %0)
  ret ptr null
}

define ptr @Counter_value(ptr %this_param) {
entry:
  %this = alloca ptr
  store ptr %this_param, ptr %this
  %0 = load ptr, ptr %this
  %1 = bitcast ptr @.str.1 to ptr
  %2 = call ptr @wolf_map_get(ptr %0, ptr %1)
  ret ptr %2
}

define ptr @BoundedCounter_new(ptr %max_param) {
entry:
  %this = alloca ptr
  %0 = call ptr @wolf_class_create(ptr null)
  store ptr %0, ptr %this
  %max = alloca ptr
  store ptr %max_param, ptr %max
  %1 = call ptr @wolf_val_int(i64 0)
  %2 = load ptr, ptr %this
  %3 = bitcast ptr @.str.1 to ptr
  call void @wolf_map_set(ptr %2, ptr %3, ptr %1)
  %4 = load ptr, ptr %max
  %5 = load ptr, ptr %this
  %6 = bitcast ptr @.str.2 to ptr
  call void @wolf_map_set(ptr %5, ptr %6, ptr %4)
  %7 = load ptr, ptr %this
  ret ptr %7
}

define ptr @BoundedCounter_increment(ptr %this_param) {
entry:
  %this = alloca ptr
  store ptr %this_param, ptr %this
  %0 = load ptr, ptr %this
  %1 = bitcast ptr @.str.1 to ptr
  %2 = call ptr @wolf_map_get(ptr %0, ptr %1)
  %3 = call ptr @wolf_val_int(i64 1)
  %4 = call i64 @wolf_val_to_i64(ptr %2)
  %5 = call i64 @wolf_val_to_i64(ptr %3)
  %6 = add i64 %4, %5
  %7 = call ptr @wolf_val_int(i64 %6)
  %next = alloca ptr
  store ptr %7, ptr %next
  %8 = load ptr, ptr %next
  %9 = load ptr, ptr %this
  %10 = bitcast ptr @.str.2 to ptr
  %11 = call ptr @wolf_map_get(ptr %9, ptr %10)
  %12 = call i64 @wolf_val_to_i64(ptr %8)
  %13 = call i64 @wolf_val_to_i64(ptr %11)
  %14 = icmp sle i64 %12, %13
  %15 = zext i1 %14 to i32
  %16 = call ptr @wolf_val_bool(i32 %15)
  %17 = call i1 @wolf_val_truthy(ptr %16)
  br i1 %17, label %if.then0, label %if.end1
if.then0:
  %18 = load ptr, ptr %next
  %19 = load ptr, ptr %this
  %20 = bitcast ptr @.str.1 to ptr
  call void @wolf_map_set(ptr %19, ptr %20, ptr %18)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @BoundedCounter_value(ptr %this_param) {
entry:
  %this = alloca ptr
  store ptr %this_param, ptr %this
  %0 = load ptr, ptr %this
  %1 = bitcast ptr @.str.1 to ptr
  %2 = call ptr @wolf_map_get(ptr %0, ptr %1)
  ret ptr %2
}

define ptr @NamedEntity_new(ptr %id_param, ptr %name_param) {
entry:
  %this = alloca ptr
  %0 = call ptr @wolf_class_create(ptr null)
  store ptr %0, ptr %this
  %id = alloca ptr
  store ptr %id_param, ptr %id
  %name = alloca ptr
  store ptr %name_param, ptr %name
  %1 = load ptr, ptr %id
  %2 = load ptr, ptr %this
  %3 = bitcast ptr @.str.3 to ptr
  call void @wolf_map_set(ptr %2, ptr %3, ptr %1)
  %4 = load ptr, ptr %name
  %5 = load ptr, ptr %this
  %6 = bitcast ptr @.str.4 to ptr
  call void @wolf_map_set(ptr %5, ptr %6, ptr %4)
  %7 = load ptr, ptr %this
  ret ptr %7
}

define ptr @NamedEntity_label(ptr %this_param) {
entry:
  %this = alloca ptr
  store ptr %this_param, ptr %this
  %0 = load ptr, ptr %this
  %1 = bitcast ptr @.str.4 to ptr
  %2 = call ptr @wolf_map_get(ptr %0, ptr %1)
  %n = alloca ptr
  store ptr %2, ptr %n
  %3 = load ptr, ptr %this
  %4 = bitcast ptr @.str.3 to ptr
  %5 = call ptr @wolf_map_get(ptr %3, ptr %4)
  %i = alloca ptr
  store ptr %5, ptr %i
  %6 = bitcast ptr @.str.5 to ptr
  ret ptr %6
}

define ptr @test_counter_basic() {
entry:
  %0 = call ptr @Counter_new()
  %c = alloca ptr
  store ptr %0, ptr %c
  %1 = load ptr, ptr %c
  %2 = call ptr @Counter_increment(ptr %1)
  %3 = load ptr, ptr %c
  %4 = call ptr @Counter_increment(ptr %3)
  %5 = load ptr, ptr %c
  %6 = call ptr @Counter_value(ptr %5)
  %val = alloca ptr
  store ptr %6, ptr %val
  %7 = load ptr, ptr %val
  %8 = call ptr @wolf_val_int(i64 2)
  %9 = call i64 @wolf_equals(ptr %7, ptr %8)
  %10 = icmp eq i64 %9, 0
  %11 = zext i1 %10 to i32
  %12 = call ptr @wolf_val_bool(i32 %11)
  %13 = call i1 @wolf_val_truthy(ptr %12)
  br i1 %13, label %if.then0, label %if.end1
if.then0:
  %14 = bitcast ptr @.str.6 to ptr
  %15 = call ptr @wolf_error(ptr %14)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_counter_triple_increment() {
entry:
  %0 = call ptr @Counter_new()
  %c = alloca ptr
  store ptr %0, ptr %c
  %1 = load ptr, ptr %c
  %2 = call ptr @Counter_increment(ptr %1)
  %3 = load ptr, ptr %c
  %4 = call ptr @Counter_increment(ptr %3)
  %5 = load ptr, ptr %c
  %6 = call ptr @Counter_increment(ptr %5)
  %7 = load ptr, ptr %c
  %8 = call ptr @Counter_value(ptr %7)
  %val = alloca ptr
  store ptr %8, ptr %val
  %9 = load ptr, ptr %val
  %10 = call ptr @wolf_val_int(i64 3)
  %11 = call i64 @wolf_equals(ptr %9, ptr %10)
  %12 = icmp eq i64 %11, 0
  %13 = zext i1 %12 to i32
  %14 = call ptr @wolf_val_bool(i32 %13)
  %15 = call i1 @wolf_val_truthy(ptr %14)
  br i1 %15, label %if.then0, label %if.end1
if.then0:
  %16 = bitcast ptr @.str.7 to ptr
  %17 = call ptr @wolf_error(ptr %16)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_counter_decrement() {
entry:
  %0 = call ptr @Counter_new()
  %c = alloca ptr
  store ptr %0, ptr %c
  %1 = load ptr, ptr %c
  %2 = call ptr @Counter_increment(ptr %1)
  %3 = load ptr, ptr %c
  %4 = call ptr @Counter_increment(ptr %3)
  %5 = load ptr, ptr %c
  %6 = call ptr @Counter_increment(ptr %5)
  %7 = load ptr, ptr %c
  %8 = call ptr @Counter_decrement(ptr %7)
  %9 = load ptr, ptr %c
  %10 = call ptr @Counter_value(ptr %9)
  %val = alloca ptr
  store ptr %10, ptr %val
  %11 = load ptr, ptr %val
  %12 = call ptr @wolf_val_int(i64 2)
  %13 = call i64 @wolf_equals(ptr %11, ptr %12)
  %14 = icmp eq i64 %13, 0
  %15 = zext i1 %14 to i32
  %16 = call ptr @wolf_val_bool(i32 %15)
  %17 = call i1 @wolf_val_truthy(ptr %16)
  br i1 %17, label %if.then0, label %if.end1
if.then0:
  %18 = bitcast ptr @.str.8 to ptr
  %19 = call ptr @wolf_error(ptr %18)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_counter_reset() {
entry:
  %0 = call ptr @Counter_new()
  %c = alloca ptr
  store ptr %0, ptr %c
  %1 = load ptr, ptr %c
  %2 = call ptr @Counter_increment(ptr %1)
  %3 = load ptr, ptr %c
  %4 = call ptr @Counter_increment(ptr %3)
  %5 = load ptr, ptr %c
  %6 = call ptr @Counter_reset(ptr %5)
  %7 = load ptr, ptr %c
  %8 = call ptr @Counter_value(ptr %7)
  %val = alloca ptr
  store ptr %8, ptr %val
  %9 = load ptr, ptr %val
  %10 = call ptr @wolf_val_int(i64 0)
  %11 = call i64 @wolf_equals(ptr %9, ptr %10)
  %12 = icmp eq i64 %11, 0
  %13 = zext i1 %12 to i32
  %14 = call ptr @wolf_val_bool(i32 %13)
  %15 = call i1 @wolf_val_truthy(ptr %14)
  br i1 %15, label %if.then0, label %if.end1
if.then0:
  %16 = bitcast ptr @.str.9 to ptr
  %17 = call ptr @wolf_error(ptr %16)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_counter_independence() {
entry:
  %0 = call ptr @Counter_new()
  %c1 = alloca ptr
  store ptr %0, ptr %c1
  %1 = call ptr @Counter_new()
  %c2 = alloca ptr
  store ptr %1, ptr %c2
  %2 = load ptr, ptr %c1
  %3 = call ptr @Counter_increment(ptr %2)
  %4 = load ptr, ptr %c1
  %5 = call ptr @Counter_increment(ptr %4)
  %6 = load ptr, ptr %c2
  %7 = call ptr @Counter_increment(ptr %6)
  %8 = load ptr, ptr %c1
  %9 = call ptr @Counter_value(ptr %8)
  %v1 = alloca ptr
  store ptr %9, ptr %v1
  %10 = load ptr, ptr %c2
  %11 = call ptr @Counter_value(ptr %10)
  %v2 = alloca ptr
  store ptr %11, ptr %v2
  %12 = load ptr, ptr %v1
  %13 = call ptr @wolf_val_int(i64 2)
  %14 = call i64 @wolf_equals(ptr %12, ptr %13)
  %15 = icmp eq i64 %14, 0
  %16 = zext i1 %15 to i32
  %17 = call ptr @wolf_val_bool(i32 %16)
  %18 = call i1 @wolf_val_truthy(ptr %17)
  br i1 %18, label %if.then0, label %if.end1
if.then0:
  %19 = bitcast ptr @.str.10 to ptr
  %20 = call ptr @wolf_error(ptr %19)
  br label %if.end1
if.end1:
  %21 = load ptr, ptr %v2
  %22 = call ptr @wolf_val_int(i64 1)
  %23 = call i64 @wolf_equals(ptr %21, ptr %22)
  %24 = icmp eq i64 %23, 0
  %25 = zext i1 %24 to i32
  %26 = call ptr @wolf_val_bool(i32 %25)
  %27 = call i1 @wolf_val_truthy(ptr %26)
  br i1 %27, label %if.then2, label %if.end3
if.then2:
  %28 = bitcast ptr @.str.11 to ptr
  %29 = call ptr @wolf_error(ptr %28)
  br label %if.end3
if.end3:
  ret ptr null
}

define ptr @test_bounded_counter() {
entry:
  %0 = call ptr @wolf_val_int(i64 3)
  %1 = call ptr @BoundedCounter_new(ptr %0)
  %bc = alloca ptr
  store ptr %1, ptr %bc
  %2 = load ptr, ptr %bc
  %3 = call ptr @BoundedCounter_increment(ptr %2)
  %4 = load ptr, ptr %bc
  %5 = call ptr @BoundedCounter_increment(ptr %4)
  %6 = load ptr, ptr %bc
  %7 = call ptr @BoundedCounter_increment(ptr %6)
  %8 = load ptr, ptr %bc
  %9 = call ptr @BoundedCounter_increment(ptr %8)
  %10 = load ptr, ptr %bc
  %11 = call ptr @BoundedCounter_increment(ptr %10)
  %12 = load ptr, ptr %bc
  %13 = call ptr @BoundedCounter_value(ptr %12)
  %val = alloca ptr
  store ptr %13, ptr %val
  %14 = load ptr, ptr %val
  %15 = call ptr @wolf_val_int(i64 3)
  %16 = call i64 @wolf_equals(ptr %14, ptr %15)
  %17 = icmp eq i64 %16, 0
  %18 = zext i1 %17 to i32
  %19 = call ptr @wolf_val_bool(i32 %18)
  %20 = call i1 @wolf_val_truthy(ptr %19)
  br i1 %20, label %if.then0, label %if.end1
if.then0:
  %21 = bitcast ptr @.str.12 to ptr
  %22 = call ptr @wolf_error(ptr %21)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_named_entity() {
entry:
  %0 = call ptr @wolf_val_int(i64 42)
  %1 = bitcast ptr @.str.13 to ptr
  %2 = call ptr @NamedEntity_new(ptr %0, ptr %1)
  %e = alloca ptr
  store ptr %2, ptr %e
  %3 = load ptr, ptr %e
  %4 = call ptr @NamedEntity_label(ptr %3)
  %label = alloca ptr
  store ptr %4, ptr %label
  %5 = load ptr, ptr %label
  %6 = bitcast ptr @.str.14 to ptr
  %7 = call i64 @wolf_equals(ptr %5, ptr %6)
  %8 = icmp eq i64 %7, 0
  %9 = zext i1 %8 to i32
  %10 = call ptr @wolf_val_bool(i32 %9)
  %11 = call i1 @wolf_val_truthy(ptr %10)
  br i1 %11, label %if.then0, label %if.end1
if.then0:
  %12 = bitcast ptr @.str.15 to ptr
  %13 = call ptr @wolf_error(ptr %12)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @add(ptr %a_param, ptr %b_param) {
entry:
  %a = alloca ptr
  store ptr %a_param, ptr %a
  %b = alloca ptr
  store ptr %b_param, ptr %b
  %0 = load ptr, ptr %a
  %1 = load ptr, ptr %b
  %2 = call i64 @wolf_val_to_i64(ptr %0)
  %3 = call i64 @wolf_val_to_i64(ptr %1)
  %4 = add i64 %2, %3
  %5 = call ptr @wolf_val_int(i64 %4)
  ret ptr %5
}

define ptr @multiply(ptr %a_param, ptr %b_param) {
entry:
  %a = alloca ptr
  store ptr %a_param, ptr %a
  %b = alloca ptr
  store ptr %b_param, ptr %b
  %0 = load ptr, ptr %a
  %1 = load ptr, ptr %b
  %2 = call i64 @wolf_val_to_i64(ptr %0)
  %3 = call i64 @wolf_val_to_i64(ptr %1)
  %4 = mul i64 %2, %3
  %5 = call ptr @wolf_val_int(i64 %4)
  ret ptr %5
}

define ptr @greet(ptr %name_param) {
entry:
  %name = alloca ptr
  store ptr %name_param, ptr %name
  %0 = bitcast ptr @.str.16 to ptr
  %1 = load ptr, ptr %name
  %2 = call ptr @wolf_string_concat(ptr %0, ptr %1)
  %3 = bitcast ptr @.str.17 to ptr
  %4 = call ptr @wolf_string_concat(ptr %2, ptr %3)
  ret ptr %4
}

define ptr @is_even(ptr %n_param) {
entry:
  %n = alloca ptr
  store ptr %n_param, ptr %n
  %0 = load ptr, ptr %n
  %1 = call ptr @wolf_val_int(i64 2)
  %2 = call i64 @wolf_val_to_i64(ptr %0)
  %3 = call i64 @wolf_val_to_i64(ptr %1)
  %4 = srem i64 %2, %3
  %5 = call ptr @wolf_val_int(i64 %4)
  %6 = call ptr @wolf_val_int(i64 0)
  %7 = call i64 @wolf_equals(ptr %5, ptr %6)
  %8 = trunc i64 %7 to i32
  %9 = call ptr @wolf_val_bool(i32 %8)
  ret ptr %9
}

define ptr @factorial(ptr %n_param) {
entry:
  %n = alloca ptr
  store ptr %n_param, ptr %n
  %0 = load ptr, ptr %n
  %1 = call ptr @wolf_val_int(i64 1)
  %2 = call i64 @wolf_val_to_i64(ptr %0)
  %3 = call i64 @wolf_val_to_i64(ptr %1)
  %4 = icmp sle i64 %2, %3
  %5 = zext i1 %4 to i32
  %6 = call ptr @wolf_val_bool(i32 %5)
  %7 = call i1 @wolf_val_truthy(ptr %6)
  br i1 %7, label %if.then0, label %if.end1
if.then0:
  %8 = call ptr @wolf_val_int(i64 1)
  ret ptr %8
if.end1:
  %9 = load ptr, ptr %n
  %10 = load ptr, ptr %n
  %11 = call ptr @wolf_val_int(i64 1)
  %12 = call i64 @wolf_val_to_i64(ptr %10)
  %13 = call i64 @wolf_val_to_i64(ptr %11)
  %14 = sub i64 %12, %13
  %15 = call ptr @wolf_val_int(i64 %14)
  %16 = call ptr @factorial(ptr %15)
  %17 = call i64 @wolf_val_to_i64(ptr %9)
  %18 = call i64 @wolf_val_to_i64(ptr %16)
  %19 = mul i64 %17, %18
  %20 = call ptr @wolf_val_int(i64 %19)
  ret ptr %20
}

define ptr @test_basic_add() {
entry:
  %0 = call ptr @wolf_val_int(i64 10)
  %1 = call ptr @wolf_val_int(i64 20)
  %2 = call ptr @add(ptr %0, ptr %1)
  %result = alloca ptr
  store ptr %2, ptr %result
  %3 = load ptr, ptr %result
  %4 = call ptr @wolf_val_int(i64 30)
  %5 = call i64 @wolf_equals(ptr %3, ptr %4)
  %6 = icmp eq i64 %5, 0
  %7 = zext i1 %6 to i32
  %8 = call ptr @wolf_val_bool(i32 %7)
  %9 = call i1 @wolf_val_truthy(ptr %8)
  br i1 %9, label %if.then0, label %if.end1
if.then0:
  %10 = bitcast ptr @.str.18 to ptr
  %11 = call ptr @wolf_error(ptr %10)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_add_negatives() {
entry:
  %0 = call ptr @wolf_val_int(i64 5)
  %1 = call i64 @wolf_val_to_i64(ptr %0)
  %2 = sub i64 0, %1
  %3 = call ptr @wolf_val_int(i64 %2)
  %4 = call ptr @wolf_val_int(i64 3)
  %5 = call ptr @add(ptr %3, ptr %4)
  %result = alloca ptr
  store ptr %5, ptr %result
  %6 = load ptr, ptr %result
  %7 = call ptr @wolf_val_int(i64 2)
  %8 = call i64 @wolf_val_to_i64(ptr %7)
  %9 = sub i64 0, %8
  %10 = call ptr @wolf_val_int(i64 %9)
  %11 = call i64 @wolf_equals(ptr %6, ptr %10)
  %12 = icmp eq i64 %11, 0
  %13 = zext i1 %12 to i32
  %14 = call ptr @wolf_val_bool(i32 %13)
  %15 = call i1 @wolf_val_truthy(ptr %14)
  br i1 %15, label %if.then0, label %if.end1
if.then0:
  %16 = bitcast ptr @.str.19 to ptr
  %17 = call ptr @wolf_error(ptr %16)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_add_zero() {
entry:
  %0 = call ptr @wolf_val_int(i64 0)
  %1 = call ptr @wolf_val_int(i64 0)
  %2 = call ptr @add(ptr %0, ptr %1)
  %result = alloca ptr
  store ptr %2, ptr %result
  %3 = load ptr, ptr %result
  %4 = call ptr @wolf_val_int(i64 0)
  %5 = call i64 @wolf_equals(ptr %3, ptr %4)
  %6 = icmp eq i64 %5, 0
  %7 = zext i1 %6 to i32
  %8 = call ptr @wolf_val_bool(i32 %7)
  %9 = call i1 @wolf_val_truthy(ptr %8)
  br i1 %9, label %if.then0, label %if.end1
if.then0:
  %10 = bitcast ptr @.str.20 to ptr
  %11 = call ptr @wolf_error(ptr %10)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_multiply() {
entry:
  %0 = call ptr @wolf_val_int(i64 6)
  %1 = call ptr @wolf_val_int(i64 7)
  %2 = call ptr @multiply(ptr %0, ptr %1)
  %result = alloca ptr
  store ptr %2, ptr %result
  %3 = load ptr, ptr %result
  %4 = call ptr @wolf_val_int(i64 42)
  %5 = call i64 @wolf_equals(ptr %3, ptr %4)
  %6 = icmp eq i64 %5, 0
  %7 = zext i1 %6 to i32
  %8 = call ptr @wolf_val_bool(i32 %7)
  %9 = call i1 @wolf_val_truthy(ptr %8)
  br i1 %9, label %if.then0, label %if.end1
if.then0:
  %10 = bitcast ptr @.str.21 to ptr
  %11 = call ptr @wolf_error(ptr %10)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_string_return() {
entry:
  %0 = bitcast ptr @.str.13 to ptr
  %1 = call ptr @greet(ptr %0)
  %result = alloca ptr
  store ptr %1, ptr %result
  %2 = load ptr, ptr %result
  %3 = bitcast ptr @.str.22 to ptr
  %4 = call i64 @wolf_equals(ptr %2, ptr %3)
  %5 = icmp eq i64 %4, 0
  %6 = zext i1 %5 to i32
  %7 = call ptr @wolf_val_bool(i32 %6)
  %8 = call i1 @wolf_val_truthy(ptr %7)
  br i1 %8, label %if.then0, label %if.end1
if.then0:
  %9 = bitcast ptr @.str.23 to ptr
  %10 = call ptr @wolf_error(ptr %9)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_bool_return() {
entry:
  %0 = call ptr @wolf_val_int(i64 4)
  %1 = call ptr @is_even(ptr %0)
  %even = alloca ptr
  store ptr %1, ptr %even
  %2 = load ptr, ptr %even
  %3 = call i1 @wolf_val_truthy(ptr %2)
  %4 = xor i1 %3, 1
  %5 = zext i1 %4 to i32
  %6 = call ptr @wolf_val_bool(i32 %5)
  %7 = call i1 @wolf_val_truthy(ptr %6)
  br i1 %7, label %if.then0, label %if.end1
if.then0:
  %8 = bitcast ptr @.str.24 to ptr
  %9 = call ptr @wolf_error(ptr %8)
  br label %if.end1
if.end1:
  %10 = call ptr @wolf_val_int(i64 7)
  %11 = call ptr @wolf_val_int(i64 2)
  %12 = call i64 @wolf_val_to_i64(ptr %10)
  %13 = call i64 @wolf_val_to_i64(ptr %11)
  %14 = srem i64 %12, %13
  %15 = call ptr @wolf_val_int(i64 %14)
  %mod = alloca ptr
  store ptr %15, ptr %mod
  %16 = load ptr, ptr %mod
  %17 = call ptr @wolf_val_int(i64 0)
  %18 = call i64 @wolf_equals(ptr %16, ptr %17)
  %19 = trunc i64 %18 to i32
  %20 = call ptr @wolf_val_bool(i32 %19)
  %21 = call i1 @wolf_val_truthy(ptr %20)
  br i1 %21, label %if.then2, label %if.end3
if.then2:
  %22 = bitcast ptr @.str.25 to ptr
  %23 = call ptr @wolf_error(ptr %22)
  br label %if.end3
if.end3:
  ret ptr null
}

define ptr @test_recursive_factorial() {
entry:
  %0 = call ptr @wolf_val_int(i64 5)
  %1 = call ptr @factorial(ptr %0)
  %result = alloca ptr
  store ptr %1, ptr %result
  %2 = load ptr, ptr %result
  %3 = call ptr @wolf_val_int(i64 120)
  %4 = call i64 @wolf_equals(ptr %2, ptr %3)
  %5 = icmp eq i64 %4, 0
  %6 = zext i1 %5 to i32
  %7 = call ptr @wolf_val_bool(i32 %6)
  %8 = call i1 @wolf_val_truthy(ptr %7)
  br i1 %8, label %if.then0, label %if.end1
if.then0:
  %9 = bitcast ptr @.str.26 to ptr
  %10 = call ptr @wolf_error(ptr %9)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_chained_calls() {
entry:
  %0 = call ptr @wolf_val_int(i64 2)
  %1 = call ptr @wolf_val_int(i64 3)
  %2 = call ptr @multiply(ptr %0, ptr %1)
  %3 = call ptr @wolf_val_int(i64 4)
  %4 = call ptr @wolf_val_int(i64 5)
  %5 = call ptr @multiply(ptr %3, ptr %4)
  %6 = call ptr @add(ptr %2, ptr %5)
  %result = alloca ptr
  store ptr %6, ptr %result
  %7 = load ptr, ptr %result
  %8 = call ptr @wolf_val_int(i64 26)
  %9 = call i64 @wolf_equals(ptr %7, ptr %8)
  %10 = icmp eq i64 %9, 0
  %11 = zext i1 %10 to i32
  %12 = call ptr @wolf_val_bool(i32 %11)
  %13 = call i1 @wolf_val_truthy(ptr %12)
  br i1 %13, label %if.then0, label %if.end1
if.then0:
  %14 = bitcast ptr @.str.27 to ptr
  %15 = call ptr @wolf_error(ptr %14)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_int_inference() {
entry:
  %0 = call ptr @wolf_val_int(i64 42)
  %x = alloca ptr
  store ptr %0, ptr %x
  %1 = load ptr, ptr %x
  %2 = call ptr @wolf_val_int(i64 2)
  %3 = call i64 @wolf_val_to_i64(ptr %1)
  %4 = call i64 @wolf_val_to_i64(ptr %2)
  %5 = mul i64 %3, %4
  %6 = call ptr @wolf_val_int(i64 %5)
  %doubled = alloca ptr
  store ptr %6, ptr %doubled
  %7 = load ptr, ptr %doubled
  %8 = call ptr @wolf_val_int(i64 84)
  %9 = call i64 @wolf_equals(ptr %7, ptr %8)
  %10 = icmp eq i64 %9, 0
  %11 = zext i1 %10 to i32
  %12 = call ptr @wolf_val_bool(i32 %11)
  %13 = call i1 @wolf_val_truthy(ptr %12)
  br i1 %13, label %if.then0, label %if.end1
if.then0:
  %14 = bitcast ptr @.str.28 to ptr
  %15 = call ptr @wolf_error(ptr %14)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_float_inference() {
entry:
  %0 = fadd double 0.0, 3.14
  %1 = call ptr @wolf_val_float(double %0)
  %y = alloca ptr
  store ptr %1, ptr %y
  %2 = load ptr, ptr %y
  %3 = fadd double 0.0, 2.0
  %4 = call ptr @wolf_val_float(double %3)
  %5 = call i64 @wolf_val_to_i64(ptr %2)
  %6 = call i64 @wolf_val_to_i64(ptr %4)
  %7 = mul i64 %5, %6
  %8 = call ptr @wolf_val_int(i64 %7)
  %scaled = alloca ptr
  store ptr %8, ptr %scaled
  %9 = load ptr, ptr %scaled
  %10 = fadd double 0.0, 6.0
  %11 = call ptr @wolf_val_float(double %10)
  %12 = call i64 @wolf_val_to_i64(ptr %9)
  %13 = call i64 @wolf_val_to_i64(ptr %11)
  %14 = icmp sle i64 %12, %13
  %15 = zext i1 %14 to i32
  %16 = call ptr @wolf_val_bool(i32 %15)
  %17 = call i1 @wolf_val_truthy(ptr %16)
  br i1 %17, label %if.then0, label %if.end1
if.then0:
  %18 = bitcast ptr @.str.29 to ptr
  %19 = call ptr @wolf_error(ptr %18)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_string_inference() {
entry:
  %0 = bitcast ptr @.str.30 to ptr
  %z = alloca ptr
  store ptr %0, ptr %z
  %1 = load ptr, ptr %z
  %2 = call ptr @wolf_strtoupper(ptr %1)
  %upper = alloca ptr
  store ptr %2, ptr %upper
  %3 = load ptr, ptr %upper
  %4 = bitcast ptr @.str.31 to ptr
  %5 = call i64 @wolf_equals(ptr %3, ptr %4)
  %6 = icmp eq i64 %5, 0
  %7 = zext i1 %6 to i32
  %8 = call ptr @wolf_val_bool(i32 %7)
  %9 = call i1 @wolf_val_truthy(ptr %8)
  br i1 %9, label %if.then0, label %if.end1
if.then0:
  %10 = bitcast ptr @.str.32 to ptr
  %11 = call ptr @wolf_error(ptr %10)
  br label %if.end1
if.end1:
  ret ptr null
}

define ptr @test_bool_inference() {
entry:
  %0 = add i1 0, 1
  %1 = call ptr @wolf_val_bool(i32 0)  ; WARN: fallback box_bool
  %flag = alloca ptr
  store ptr %1, ptr %flag
  %2 = load ptr, ptr %flag
  %3 = call i1 @wolf_val_truthy(ptr %2)
  %4 = xor i1 %3, 1
  %5 = zext i1 %4 to i32
  %6 = call ptr @wolf_val_bool(i32 %5)
  %7 = call i1 @wolf_val_truthy(ptr %6)
  br i1 %7, label %if.then0, label %if.end1
if.then0:
  %8 = bitcast ptr @.str.33 to ptr
  %9 = call ptr @wolf_error(ptr %8)
  br label %if.end1
if.end1:
  %10 = add i1 0, 0
  %11 = call ptr @wolf_val_bool(i32 0)  ; WARN: fallback box_bool
  %flag2 = alloca ptr
  store ptr %11, ptr %flag2
  %12 = load ptr, ptr %flag2
  %13 = call i1 @wolf_val_truthy(ptr %12)
  br i1 %13, label %if.then2, label %if.end3
if.then2:
  %14 = bitcast ptr @.str.34 to ptr
  %15 = call ptr @wolf_error(ptr %14)
  br label %if.end3
if.end3:
  ret ptr null
}

define ptr @test_type_isolation() {
entry:
  %0 = call ptr @wolf_val_int(i64 100)
  %n = alloca ptr
  store ptr %0, ptr %n
  %1 = bitcast ptr @.str.35 to ptr
  %s = alloca ptr
  store ptr %1, ptr %s
  %2 = load ptr, ptr %s
  %3 = bitcast ptr @.str.36 to ptr
  %4 = call ptr @wolf_string_concat(ptr %2, ptr %3)
  %combined = alloca ptr
  store ptr %4, ptr %combined
  %5 = load ptr, ptr %combined
  %6 = bitcast ptr @.str.37 to ptr
  %7 = call i64 @wolf_equals(ptr %5, ptr %6)
  %8 = icmp eq i64 %7, 0
  %9 = zext i1 %8 to i32
  %10 = call ptr @wolf_val_bool(i32 %9)
  %11 = call i1 @wolf_val_truthy(ptr %10)
  br i1 %11, label %if.then0, label %if.end1
if.then0:
  %12 = bitcast ptr @.str.38 to ptr
  %13 = call ptr @wolf_error(ptr %12)
  br label %if.end1
if.end1:
  %14 = load ptr, ptr %n
  %15 = call ptr @wolf_val_int(i64 1)
  %16 = call i64 @wolf_val_to_i64(ptr %14)
  %17 = call i64 @wolf_val_to_i64(ptr %15)
  %18 = add i64 %16, %17
  %19 = call ptr @wolf_val_int(i64 %18)
  %n2 = alloca ptr
  store ptr %19, ptr %n2
  %20 = load ptr, ptr %n2
  %21 = call ptr @wolf_val_int(i64 101)
  %22 = call i64 @wolf_equals(ptr %20, ptr %21)
  %23 = icmp eq i64 %22, 0
  %24 = zext i1 %23 to i32
  %25 = call ptr @wolf_val_bool(i32 %24)
  %26 = call i1 @wolf_val_truthy(ptr %25)
  br i1 %26, label %if.then2, label %if.end3
if.then2:
  %27 = bitcast ptr @.str.39 to ptr
  %28 = call ptr @wolf_error(ptr %27)
  br label %if.end3
if.end3:
  ret ptr null
}

define i64 @main() {
entry:
  %0 = call ptr @wolf_val_int(i64 0)
  %passed = alloca ptr
  store ptr %0, ptr %passed
  %1 = call ptr @wolf_val_int(i64 0)
  %failed = alloca ptr
  store ptr %1, ptr %failed
  %2 = call ptr @test_counter_basic()
  %3 = call ptr @test_counter_triple_increment()
  %4 = call ptr @test_counter_decrement()
  %5 = call ptr @test_counter_reset()
  %6 = call ptr @test_counter_independence()
  %7 = call ptr @test_bounded_counter()
  %8 = call ptr @test_named_entity()
  %9 = bitcast ptr @.str.40 to ptr
  call void @wolf_print_str(ptr %9)
  call void @wolf_println()
  %10 = call ptr @test_basic_add()
  %11 = call ptr @test_add_negatives()
  %12 = call ptr @test_add_zero()
  %13 = call ptr @test_multiply()
  %14 = call ptr @test_string_return()
  %15 = call ptr @test_bool_return()
  %16 = call ptr @test_recursive_factorial()
  %17 = call ptr @test_chained_calls()
  %18 = bitcast ptr @.str.41 to ptr
  call void @wolf_print_str(ptr %18)
  call void @wolf_println()
  %19 = call ptr @test_int_inference()
  %20 = call ptr @test_float_inference()
  %21 = call ptr @test_string_inference()
  %22 = call ptr @test_bool_inference()
  %23 = call ptr @test_type_isolation()
  %24 = bitcast ptr @.str.42 to ptr
  call void @wolf_print_str(ptr %24)
  call void @wolf_println()
  %25 = call ptr @test_counter_basic()
  %26 = bitcast ptr @.str.43 to ptr
  call void @wolf_print_str(ptr %26)
  call void @wolf_println()
  %27 = load ptr, ptr %passed
  %28 = call ptr @wolf_val_int(i64 1)
  %29 = call i64 @wolf_val_to_i64(ptr %27)
  %30 = call i64 @wolf_val_to_i64(ptr %28)
  %31 = add i64 %29, %30
  %32 = call ptr @wolf_val_int(i64 %31)
  store ptr %32, ptr %passed
  %33 = call ptr @test_counter_triple_increment()
  %34 = bitcast ptr @.str.44 to ptr
  call void @wolf_print_str(ptr %34)
  call void @wolf_println()
  %35 = load ptr, ptr %passed
  %36 = call ptr @wolf_val_int(i64 1)
  %37 = call i64 @wolf_val_to_i64(ptr %35)
  %38 = call i64 @wolf_val_to_i64(ptr %36)
  %39 = add i64 %37, %38
  %40 = call ptr @wolf_val_int(i64 %39)
  store ptr %40, ptr %passed
  %41 = call ptr @test_counter_decrement()
  %42 = bitcast ptr @.str.45 to ptr
  call void @wolf_print_str(ptr %42)
  call void @wolf_println()
  %43 = load ptr, ptr %passed
  %44 = call ptr @wolf_val_int(i64 1)
  %45 = call i64 @wolf_val_to_i64(ptr %43)
  %46 = call i64 @wolf_val_to_i64(ptr %44)
  %47 = add i64 %45, %46
  %48 = call ptr @wolf_val_int(i64 %47)
  store ptr %48, ptr %passed
  %49 = call ptr @test_counter_reset()
  %50 = bitcast ptr @.str.46 to ptr
  call void @wolf_print_str(ptr %50)
  call void @wolf_println()
  %51 = load ptr, ptr %passed
  %52 = call ptr @wolf_val_int(i64 1)
  %53 = call i64 @wolf_val_to_i64(ptr %51)
  %54 = call i64 @wolf_val_to_i64(ptr %52)
  %55 = add i64 %53, %54
  %56 = call ptr @wolf_val_int(i64 %55)
  store ptr %56, ptr %passed
  %57 = call ptr @test_counter_independence()
  %58 = bitcast ptr @.str.47 to ptr
  call void @wolf_print_str(ptr %58)
  call void @wolf_println()
  %59 = load ptr, ptr %passed
  %60 = call ptr @wolf_val_int(i64 1)
  %61 = call i64 @wolf_val_to_i64(ptr %59)
  %62 = call i64 @wolf_val_to_i64(ptr %60)
  %63 = add i64 %61, %62
  %64 = call ptr @wolf_val_int(i64 %63)
  store ptr %64, ptr %passed
  %65 = call ptr @test_bounded_counter()
  %66 = bitcast ptr @.str.48 to ptr
  call void @wolf_print_str(ptr %66)
  call void @wolf_println()
  %67 = load ptr, ptr %passed
  %68 = call ptr @wolf_val_int(i64 1)
  %69 = call i64 @wolf_val_to_i64(ptr %67)
  %70 = call i64 @wolf_val_to_i64(ptr %68)
  %71 = add i64 %69, %70
  %72 = call ptr @wolf_val_int(i64 %71)
  store ptr %72, ptr %passed
  %73 = call ptr @test_named_entity()
  %74 = bitcast ptr @.str.49 to ptr
  call void @wolf_print_str(ptr %74)
  call void @wolf_println()
  %75 = load ptr, ptr %passed
  %76 = call ptr @wolf_val_int(i64 1)
  %77 = call i64 @wolf_val_to_i64(ptr %75)
  %78 = call i64 @wolf_val_to_i64(ptr %76)
  %79 = add i64 %77, %78
  %80 = call ptr @wolf_val_int(i64 %79)
  store ptr %80, ptr %passed
  %81 = call ptr @test_basic_add()
  %82 = bitcast ptr @.str.50 to ptr
  call void @wolf_print_str(ptr %82)
  call void @wolf_println()
  %83 = load ptr, ptr %passed
  %84 = call ptr @wolf_val_int(i64 1)
  %85 = call i64 @wolf_val_to_i64(ptr %83)
  %86 = call i64 @wolf_val_to_i64(ptr %84)
  %87 = add i64 %85, %86
  %88 = call ptr @wolf_val_int(i64 %87)
  store ptr %88, ptr %passed
  %89 = call ptr @test_add_negatives()
  %90 = bitcast ptr @.str.51 to ptr
  call void @wolf_print_str(ptr %90)
  call void @wolf_println()
  %91 = load ptr, ptr %passed
  %92 = call ptr @wolf_val_int(i64 1)
  %93 = call i64 @wolf_val_to_i64(ptr %91)
  %94 = call i64 @wolf_val_to_i64(ptr %92)
  %95 = add i64 %93, %94
  %96 = call ptr @wolf_val_int(i64 %95)
  store ptr %96, ptr %passed
  %97 = call ptr @test_add_zero()
  %98 = bitcast ptr @.str.52 to ptr
  call void @wolf_print_str(ptr %98)
  call void @wolf_println()
  %99 = load ptr, ptr %passed
  %100 = call ptr @wolf_val_int(i64 1)
  %101 = call i64 @wolf_val_to_i64(ptr %99)
  %102 = call i64 @wolf_val_to_i64(ptr %100)
  %103 = add i64 %101, %102
  %104 = call ptr @wolf_val_int(i64 %103)
  store ptr %104, ptr %passed
  %105 = call ptr @test_multiply()
  %106 = bitcast ptr @.str.53 to ptr
  call void @wolf_print_str(ptr %106)
  call void @wolf_println()
  %107 = load ptr, ptr %passed
  %108 = call ptr @wolf_val_int(i64 1)
  %109 = call i64 @wolf_val_to_i64(ptr %107)
  %110 = call i64 @wolf_val_to_i64(ptr %108)
  %111 = add i64 %109, %110
  %112 = call ptr @wolf_val_int(i64 %111)
  store ptr %112, ptr %passed
  %113 = call ptr @test_string_return()
  %114 = bitcast ptr @.str.54 to ptr
  call void @wolf_print_str(ptr %114)
  call void @wolf_println()
  %115 = load ptr, ptr %passed
  %116 = call ptr @wolf_val_int(i64 1)
  %117 = call i64 @wolf_val_to_i64(ptr %115)
  %118 = call i64 @wolf_val_to_i64(ptr %116)
  %119 = add i64 %117, %118
  %120 = call ptr @wolf_val_int(i64 %119)
  store ptr %120, ptr %passed
  %121 = call ptr @test_bool_return()
  %122 = bitcast ptr @.str.55 to ptr
  call void @wolf_print_str(ptr %122)
  call void @wolf_println()
  %123 = load ptr, ptr %passed
  %124 = call ptr @wolf_val_int(i64 1)
  %125 = call i64 @wolf_val_to_i64(ptr %123)
  %126 = call i64 @wolf_val_to_i64(ptr %124)
  %127 = add i64 %125, %126
  %128 = call ptr @wolf_val_int(i64 %127)
  store ptr %128, ptr %passed
  %129 = call ptr @test_recursive_factorial()
  %130 = bitcast ptr @.str.56 to ptr
  call void @wolf_print_str(ptr %130)
  call void @wolf_println()
  %131 = load ptr, ptr %passed
  %132 = call ptr @wolf_val_int(i64 1)
  %133 = call i64 @wolf_val_to_i64(ptr %131)
  %134 = call i64 @wolf_val_to_i64(ptr %132)
  %135 = add i64 %133, %134
  %136 = call ptr @wolf_val_int(i64 %135)
  store ptr %136, ptr %passed
  %137 = call ptr @test_chained_calls()
  %138 = bitcast ptr @.str.57 to ptr
  call void @wolf_print_str(ptr %138)
  call void @wolf_println()
  %139 = load ptr, ptr %passed
  %140 = call ptr @wolf_val_int(i64 1)
  %141 = call i64 @wolf_val_to_i64(ptr %139)
  %142 = call i64 @wolf_val_to_i64(ptr %140)
  %143 = add i64 %141, %142
  %144 = call ptr @wolf_val_int(i64 %143)
  store ptr %144, ptr %passed
  %145 = call ptr @test_int_inference()
  %146 = bitcast ptr @.str.58 to ptr
  call void @wolf_print_str(ptr %146)
  call void @wolf_println()
  %147 = load ptr, ptr %passed
  %148 = call ptr @wolf_val_int(i64 1)
  %149 = call i64 @wolf_val_to_i64(ptr %147)
  %150 = call i64 @wolf_val_to_i64(ptr %148)
  %151 = add i64 %149, %150
  %152 = call ptr @wolf_val_int(i64 %151)
  store ptr %152, ptr %passed
  %153 = call ptr @test_float_inference()
  %154 = bitcast ptr @.str.59 to ptr
  call void @wolf_print_str(ptr %154)
  call void @wolf_println()
  %155 = load ptr, ptr %passed
  %156 = call ptr @wolf_val_int(i64 1)
  %157 = call i64 @wolf_val_to_i64(ptr %155)
  %158 = call i64 @wolf_val_to_i64(ptr %156)
  %159 = add i64 %157, %158
  %160 = call ptr @wolf_val_int(i64 %159)
  store ptr %160, ptr %passed
  %161 = call ptr @test_string_inference()
  %162 = bitcast ptr @.str.60 to ptr
  call void @wolf_print_str(ptr %162)
  call void @wolf_println()
  %163 = load ptr, ptr %passed
  %164 = call ptr @wolf_val_int(i64 1)
  %165 = call i64 @wolf_val_to_i64(ptr %163)
  %166 = call i64 @wolf_val_to_i64(ptr %164)
  %167 = add i64 %165, %166
  %168 = call ptr @wolf_val_int(i64 %167)
  store ptr %168, ptr %passed
  %169 = call ptr @test_bool_inference()
  %170 = bitcast ptr @.str.61 to ptr
  call void @wolf_print_str(ptr %170)
  call void @wolf_println()
  %171 = load ptr, ptr %passed
  %172 = call ptr @wolf_val_int(i64 1)
  %173 = call i64 @wolf_val_to_i64(ptr %171)
  %174 = call i64 @wolf_val_to_i64(ptr %172)
  %175 = add i64 %173, %174
  %176 = call ptr @wolf_val_int(i64 %175)
  store ptr %176, ptr %passed
  %177 = call ptr @test_type_isolation()
  %178 = bitcast ptr @.str.62 to ptr
  call void @wolf_print_str(ptr %178)
  call void @wolf_println()
  %179 = load ptr, ptr %passed
  %180 = call ptr @wolf_val_int(i64 1)
  %181 = call i64 @wolf_val_to_i64(ptr %179)
  %182 = call i64 @wolf_val_to_i64(ptr %180)
  %183 = add i64 %181, %182
  %184 = call ptr @wolf_val_int(i64 %183)
  store ptr %184, ptr %passed
  %185 = bitcast ptr @.str.63 to ptr
  %186 = load ptr, ptr %passed
  %187 = call ptr @wolf_string_concat(ptr %185, ptr %186)
  %188 = bitcast ptr @.str.64 to ptr
  %189 = call ptr @wolf_string_concat(ptr %187, ptr %188)
  %190 = load ptr, ptr %failed
  %191 = call ptr @wolf_string_concat(ptr %189, ptr %190)
  %192 = bitcast ptr @.str.65 to ptr
  %193 = call ptr @wolf_string_concat(ptr %191, ptr %192)
  call void @wolf_print_str(ptr %193)
  call void @wolf_println()
  %194 = load ptr, ptr %failed
  %195 = call ptr @wolf_val_int(i64 0)
  %196 = call i64 @wolf_val_to_i64(ptr %194)
  %197 = call i64 @wolf_val_to_i64(ptr %195)
  %198 = icmp sgt i64 %196, %197
  %199 = zext i1 %198 to i32
  %200 = call ptr @wolf_val_bool(i32 %199)
  %201 = call i1 @wolf_val_truthy(ptr %200)
  br i1 %201, label %if.then0, label %if.end1
if.then0:
  %202 = call ptr @wolf_val_int(i64 1)
  %203 = call ptr @wolf_system_exit_ptr(ptr %202)
  br label %if.end1
if.end1:
  ret i64 0
}

@.str.1 = private unnamed_addr constant [6 x i8] c"count\00"
@.str.2 = private unnamed_addr constant [4 x i8] c"max\00"
@.str.3 = private unnamed_addr constant [3 x i8] c"id\00"
@.str.4 = private unnamed_addr constant [5 x i8] c"name\00"
@.str.5 = private unnamed_addr constant [10 x i8] c"{$n}#{$i}\00"
@.str.6 = private unnamed_addr constant [58 x i8] c"T1-03: Counter after 2 increments expected 2, got: {$val}\00"
@.str.7 = private unnamed_addr constant [58 x i8] c"T1-03: Counter after 3 increments expected 3, got: {$val}\00"
@.str.8 = private unnamed_addr constant [72 x i8] c"T1-03: Counter after 3 increments + 1 decrement expected 2, got: {$val}\00"
@.str.9 = private unnamed_addr constant [51 x i8] c"T1-03: Counter after reset expected 0, got: {$val}\00"
@.str.10 = private unnamed_addr constant [33 x i8] c"T1-03: c1 expected 2, got: {$v1}\00"
@.str.11 = private unnamed_addr constant [33 x i8] c"T1-03: c2 expected 1, got: {$v2}\00"
@.str.12 = private unnamed_addr constant [68 x i8] c"T1-03: BoundedCounter(3) after 5 increments expected 3, got: {$val}\00"
@.str.13 = private unnamed_addr constant [5 x i8] c"Wolf\00"
@.str.14 = private unnamed_addr constant [8 x i8] c"Wolf#42\00"
@.str.15 = private unnamed_addr constant [59 x i8] c"T1-03: NamedEntity label expected 'Wolf#42', got: {$label}\00"
@.str.16 = private unnamed_addr constant [8 x i8] c"Hello, \00"
@.str.17 = private unnamed_addr constant [2 x i8] c"!\00"
@.str.18 = private unnamed_addr constant [46 x i8] c"T1-02: add(10,20) expected 30, got: {$result}\00"
@.str.19 = private unnamed_addr constant [45 x i8] c"T1-02: add(-5,3) expected -2, got: {$result}\00"
@.str.20 = private unnamed_addr constant [43 x i8] c"T1-02: add(0,0) expected 0, got: {$result}\00"
@.str.21 = private unnamed_addr constant [49 x i8] c"T1-02: multiply(6,7) expected 42, got: {$result}\00"
@.str.22 = private unnamed_addr constant [13 x i8] c"Hello, Wolf!\00"
@.str.23 = private unnamed_addr constant [59 x i8] c"T1-02: greet(Wolf) expected 'Hello, Wolf!', got: {$result}\00"
@.str.24 = private unnamed_addr constant [33 x i8] c"T1-02: is_even(4) should be true\00"
@.str.25 = private unnamed_addr constant [45 x i8] c"T1-02: 7 mod 2 should not be 0 (got: {$mod})\00"
@.str.26 = private unnamed_addr constant [49 x i8] c"T1-02: factorial(5) expected 120, got: {$result}\00"
@.str.27 = private unnamed_addr constant [49 x i8] c"T1-02: chained calls expected 26, got: {$result}\00"
@.str.28 = private unnamed_addr constant [61 x i8] c"T1-01: int inference failed — expected 84, got: {$doubled}\00"
@.str.29 = private unnamed_addr constant [49 x i8] c"T1-01: float inference failed — expected > 6.0\00"
@.str.30 = private unnamed_addr constant [6 x i8] c"hello\00"
@.str.31 = private unnamed_addr constant [6 x i8] c"HELLO\00"
@.str.32 = private unnamed_addr constant [65 x i8] c"T1-01: string inference failed — expected HELLO, got: {$upper}\00"
@.str.33 = private unnamed_addr constant [53 x i8] c"T1-01: bool inference failed — flag should be true\00"
@.str.34 = private unnamed_addr constant [55 x i8] c"T1-01: bool inference failed — flag2 should be false\00"
@.str.35 = private unnamed_addr constant [4 x i8] c"100\00"
@.str.36 = private unnamed_addr constant [3 x i8] c"px\00"
@.str.37 = private unnamed_addr constant [6 x i8] c"100px\00"
@.str.38 = private unnamed_addr constant [66 x i8] c"T1-01: type isolation failed — expected 100px, got: {$combined}\00"
@.str.39 = private unnamed_addr constant [58 x i8] c"T1-01: int arithmetic failed — expected 101, got: {$n2}\00"
@.str.40 = private unnamed_addr constant [41 x i8] c"ALL TESTS PASSED IN t1_classes_test.wolf\00"
@.str.41 = private unnamed_addr constant [43 x i8] c"ALL TESTS PASSED IN t1_functions_test.wolf\00"
@.str.42 = private unnamed_addr constant [48 x i8] c"ALL TESTS PASSED IN t1_type_inference_test.wolf\00"
@.str.43 = private unnamed_addr constant [23 x i8] c"✅ test_counter_basic\00"
@.str.44 = private unnamed_addr constant [34 x i8] c"✅ test_counter_triple_increment\00"
@.str.45 = private unnamed_addr constant [27 x i8] c"✅ test_counter_decrement\00"
@.str.46 = private unnamed_addr constant [23 x i8] c"✅ test_counter_reset\00"
@.str.47 = private unnamed_addr constant [30 x i8] c"✅ test_counter_independence\00"
@.str.48 = private unnamed_addr constant [25 x i8] c"✅ test_bounded_counter\00"
@.str.49 = private unnamed_addr constant [22 x i8] c"✅ test_named_entity\00"
@.str.50 = private unnamed_addr constant [19 x i8] c"✅ test_basic_add\00"
@.str.51 = private unnamed_addr constant [23 x i8] c"✅ test_add_negatives\00"
@.str.52 = private unnamed_addr constant [18 x i8] c"✅ test_add_zero\00"
@.str.53 = private unnamed_addr constant [18 x i8] c"✅ test_multiply\00"
@.str.54 = private unnamed_addr constant [23 x i8] c"✅ test_string_return\00"
@.str.55 = private unnamed_addr constant [21 x i8] c"✅ test_bool_return\00"
@.str.56 = private unnamed_addr constant [29 x i8] c"✅ test_recursive_factorial\00"
@.str.57 = private unnamed_addr constant [23 x i8] c"✅ test_chained_calls\00"
@.str.58 = private unnamed_addr constant [23 x i8] c"✅ test_int_inference\00"
@.str.59 = private unnamed_addr constant [25 x i8] c"✅ test_float_inference\00"
@.str.60 = private unnamed_addr constant [26 x i8] c"✅ test_string_inference\00"
@.str.61 = private unnamed_addr constant [24 x i8] c"✅ test_bool_inference\00"
@.str.62 = private unnamed_addr constant [24 x i8] c"✅ test_type_isolation\00"
@.str.63 = private unnamed_addr constant [10 x i8] c"\nTests: \00"
@.str.64 = private unnamed_addr constant [10 x i8] c" passed, \00"
@.str.65 = private unnamed_addr constant [8 x i8] c" failed\00"

; --- External Runtime Functions ---
declare void @wolf_print_str(ptr)
declare void @wolf_println()
declare ptr @wolf_int_to_string(i64)
declare i64 @wolf_string_to_int(ptr)
declare ptr @wolf_string_concat(ptr, ptr)
declare i1 @streq(ptr, ptr)
declare i64 @wolf_equals(ptr, ptr)
declare ptr @wolf_array_create()
declare void @wolf_array_push(ptr, ptr)
declare i64 @wolf_array_length(ptr)
declare ptr @wolf_array_get(ptr, i64)
declare ptr @wolf_map_create()
declare void @wolf_map_set(ptr, ptr, ptr)
declare ptr @wolf_map_get(ptr, ptr)
declare ptr @wolf_class_create(ptr)
declare ptr @wolf_val_int(i64)
declare ptr @wolf_val_float(double)
declare ptr @wolf_val_bool(i32)
declare i64 @wolf_val_to_i64(ptr)
declare i1 @wolf_val_truthy(ptr)
declare i64 @wolf_method_call(...)
declare ptr @wolf_error(ptr)
declare ptr @wolf_strtoupper(ptr)
declare ptr @wolf_strtolower(ptr)
declare ptr @wolf_strlen(ptr)
declare ptr @wolf_trim(ptr)
declare ptr @wolf_substr(ptr, ptr, ptr)
declare ptr @wolf_explode(ptr, ptr)
declare ptr @wolf_implode(ptr, ptr)
declare ptr @wolf_system_exec_ptr(ptr)
declare ptr @wolf_system_exit_ptr(ptr)
declare ptr @wolf_path_join(ptr, ptr)
declare ptr @wolf_file_write_ptr(ptr, ptr)
declare ptr @wolf_file_delete_ptr(ptr)
declare ptr @wolf_file_read(ptr)
declare ptr @wolf_file_basename(ptr)
declare ptr @wolf_dir_exists_ptr(ptr)
declare ptr @wolf_file_exists_ptr(ptr)
declare i1 @wolf_dir_exists(ptr)
declare ptr @wolf_file_list_dir(ptr)
declare ptr @wolf_str_starts_with(ptr, ptr)
declare ptr @wolf_str_contains(ptr, ptr)
declare ptr @wolf_str_ends_with(ptr, ptr)
declare ptr @wolf_str_replace(ptr, ptr, ptr)
declare ptr @wolf_str_split(ptr, ptr)


