export function PageHeader({ title, desc }: { title: string; desc: string }) {
  return (
    <header className="page-header">
      <h1>{title}</h1>
      <p>{desc}</p>
    </header>
  )
}
