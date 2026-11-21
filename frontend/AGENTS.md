## Server state and forms

- Use TanStack Query for asynchronous server state. When introducing queries, provide a stable `QueryClient` through `QueryClientProvider`, use keys that include the data's dependencies, and invalidate affected queries after mutations.
- Use Zustand for shared client state when it needs to be accessed across components; keep server state in TanStack Query.
- Use React Hook Form for forms that need validation or submission state. Keep validation errors associated with their fields and expose submission errors to the user.

- For new or changed form fields, follow the [shadcn React Hook Form pattern](https://ui.shadcn.com/docs/forms/react-hook-form): compose `Controller` with `Field`, `FieldLabel`, and `FieldError`, and associate validation errors with their controls. Use `FieldGroup` for form layout instead of the custom `FormField` wrapper.
