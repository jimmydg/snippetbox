package main

type contextKey string

const isAuthenticatedContextKey = contextKey("isAuthenticated")
const authenticatedUserId = "authenticatedUserId"
const redirectAfterLogin = "redirectAfterLogin"
