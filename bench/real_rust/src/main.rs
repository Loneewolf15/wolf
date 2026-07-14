use axum::{
    routing::get,
    Router,
    response::Json,
};
use serde::{Deserialize, Serialize};

#[derive(Deserialize)]
struct Payload {
    user_id: i32,
    action: String,
    timestamp: i32,
}

#[derive(Serialize)]
struct Response {
    sql: String,
    math_result: i32,
}

async fn handle_request() -> Json<Response> {
    let data = r#"{"user_id": 999, "action": "login", "timestamp": 160000}"#;
    let _p: Payload = serde_json::from_str(data).unwrap();

    let mut res = 1;
    for i in 1..=20 {
        res += i * 2;
    }

    let sql = "SELECT * FROM users WHERE id = 999 LIMIT 10".to_string();

    Json(Response {
        sql,
        math_result: res,
    })
}

#[tokio::main]
async fn main() {
    let app = Router::new().route("/", get(handle_request));

    // Running on 8085 as per our sequence (Go=8081, Node=8082, Py=8083, Wolf=8084, Rust=8085)
    let listener = tokio::net::TcpListener::bind("127.0.0.1:8085").await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
