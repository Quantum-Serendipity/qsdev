# C#/.NET Conventions

- Follow .NET naming: `PascalCase` for public members, `camelCase` for locals, `_camelCase` for private fields.
- Use `async`/`await` consistently. Never block on async code with `.Result` or `.Wait()`.
- Prefer records or `readonly struct` for immutable data transfer objects.
- Use dependency injection via `IServiceCollection`. Avoid newing services directly.
- Handle errors with specific exception types, not generic `Exception`.
- Write tests with xUnit or NUnit. Use `[Theory]`/`[InlineData]` for parameterized tests.
- Use `using` statements for `IDisposable`/`IAsyncDisposable` resource cleanup.
- Keep NuGet packages locked via `packages.lock.json`.
- Enable nullable reference types (`<Nullable>enable</Nullable>`) and resolve all warnings.
- Run `dotnet build --warnaserror` and `dotnet test` before committing.
