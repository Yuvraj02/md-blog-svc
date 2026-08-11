env "local" {
  url = getenv("DATABASE_URL")
  dev = getenv("DEV_DATABASE_URL")

  migration {
    dir = "file://migrations"
  }
}

lint {
  destructive {
    error = true
  }
}
