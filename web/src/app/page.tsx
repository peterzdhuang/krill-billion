export default function Home() {
  return (
    <main className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden px-6">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 opacity-40"
        style={{
          backgroundImage:
            "radial-gradient(circle at 20% 20%, rgba(45,212,191,0.12), transparent 40%), radial-gradient(circle at 80% 70%, rgba(14,116,144,0.18), transparent 45%)",
        }}
      />
      <div className="relative z-10 max-w-2xl text-center">
        <p className="mb-3 text-sm tracking-[0.35em] text-cyan-accent/70">DEPTH PROTOCOL</p>
        <h1 className="text-6xl leading-none tracking-wide text-cyan-glow sm:text-7xl">
          KRILL BILLION
        </h1>
        <p className="mt-6 text-xl text-slate-300/90 sm:text-2xl">
          Rarity duels in the deep. Guess rarer. Deal more damage.
        </p>
        <p className="mt-10 text-lg text-slate-500">Lobby create UI lands in issue #11.</p>
      </div>
    </main>
  );
}
