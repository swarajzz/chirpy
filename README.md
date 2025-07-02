# 🐦 Chirpy — A Tiny Backend for Big Thoughts

Chirpy is a lightweight backend server for a fictional microblogging app, built entirely in **Go**. Think of it as Twitter’s much smaller, much nerdier cousin—created as a personal learning project to get hands-on with building APIs, handling authentication, and working with PostgreSQL in Go.

This is not, in fact, a real project, but instead a guided project created through a [boot.dev](https://www.boot.dev) course. It's supposed to be a faux social media where you can create a user, log in, create *Chirps*, and then list them all, including capacities to filter and sort them. It's kinda nice.

---

## 🚀 Features

* 🔐 User Registration & Login (with hashed passwords)
* 🐣 Post "chirps" (140 characters or less)
* 🚫 Built-in Censorship (no kerfuffle, sharbert, or fornax allowed 😉)
* ✨ JWT-based Authentication
* 🔁 Refresh Token system
* 🔒 Token revocation and account updates
* 📬 Webhook endpoint to simulate "premium user" upgrades
* 🧪 Health check endpoint for readiness probes

---

## 🧰 Tech Stack

* **Language**: Go 🐹
* **Database**: PostgreSQL 🐘
* **Dependencies**:
    * `github.com/google/uuid`
    * `github.com/joho/godotenv`
    * `github.com/lib/pq` (PostgreSQL driver)

---

## 📦 Getting Started

1.  **Clone the repo**

    ```bash
    git clone [https://github.com/your-username/chirpy.git](https://github.com/your-username/chirpy.git)
    cd chirpy
    ```

2.  **Setup environment**

    Create a `.env` file with the following:

    ```env
    DB_URL=postgres://<your-db-credentials>
    SECRET=<your-jwt-secret>
    POLKA_KEY=<some-magic-api-key>
    PLATFORM=DEV
    ```

3.  **Run the server**

    ```bash
    go run .
    ```

    Server will start on `http://localhost:8080`

---

## 🧪 API Overview

| Method | Endpoint | Description |
| :----- | :------- | :---------- |
| `GET` | `/api/healthz` | Health check |
| `POST` | `/api/users` | Register new user |
| `POST` | `/api/login` | Login and receive JWTs |
| `POST` | `/api/refresh` | Get new access token |
| `POST` | `/api/revoke` | Revoke refresh token |
| `GET` | `/api/chirps` | Get all chirps |
| `GET` | `/api/chirps?author_id=xyz` | Filter chirps by author |
| `POST` | `/api/chirps` | Create a chirp (auth required) |
| `GET` | `/api/chirps/{id}` | Get a specific chirp |
| `DELETE` | `/api/chirps/{id}` | Delete a chirp (auth required) |
| `PUT` | `/api/users` | Update email/password |
| `POST` | `/api/polka/webhooks` | Handle premium user upgrades |

---

### 🔧 Admin Endpoints:

* `GET /admin/metrics`: View file server hit count
* `POST /admin/reset`: Reset user DB + metrics (only in DEV mode)

---

## 🤖 Censorship Bot

To keep Chirpy civil, certain words are auto-censored. Example:

```arduino
"kerfuffle at the park" → "**** at the park"
```

---

## 🧠 Why I Built This 

* Learn Go in a hands-on way
* Get comfortable with HTTP servers, routing, and middleware
* Work with PostgreSQL and understand basic DB ops in Go
* Implement secure auth patterns like JWT and refresh tokens

---

## 🪵 Sample Logs 

* Learn Go in a hands-on way
    ```bash
    Server is starting on :8080
    dburl postgres://...
    ```
---

## 🎯 Next Steps (Maybe?)

* Add WebSockets for real-time chirping
* Build a simple frontend in React/Svelte
* Add pagination for chirp feeds
* Emoji reactions and chirp replies 😎

---

## 📝 License

MIT. Use it, fork it, chirp it.

---

## 🙌 Final Thoughts

Thanks for checking out Chirpy!
I built this project to learn Go and had a blast doing it.
If you’re also learning Go, feel free to explore, extend, and chirp away 🚀