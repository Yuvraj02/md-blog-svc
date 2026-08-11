# Blog Service

One module owns Article end-to-end (model, store, draft + publish use cases, transport):

```text
internal/
├── app/
│   └── article/
│       ├── models/
│       │   └── article_model.go
│       ├── store.go / store_gorm.go
│       ├── draft.go / publish.go
│       └── transport/
├── server/grpc.go
├── config/
└── infrastructure/
```

See [../../docs/architecture.md](../../docs/architecture.md).
