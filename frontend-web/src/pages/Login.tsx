import { createSignal } from 'solid-js';
import { login } from '../services/auth';

export default function Login() {
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [error, setError] = createSignal('');
  const [loading, setLoading] = createSignal(false);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(email(), password());
      window.location.href = '/';
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class="min-h-screen bg-black flex items-center justify-center p-4">
      <div class="w-full max-w-md bg-zinc-900 border border-zinc-700 p-8 font-mono shadow-[0_0_50px_rgba(0,0,0,0.5)]">
        <div class="mb-8 border-b border-zinc-700 pb-4">
          <h1 class="text-2xl font-black text-white tracking-tighter uppercase italic">
            OBLIVRA <span class="bg-red-600 px-1 not-italic text-black">ENTERPRISE</span>
          </h1>
          <p class="text-zinc-500 text-xs mt-1 uppercase tracking-widest">Headless Access Portal v1.0.0-MVP</p>
        </div>

        <form onSubmit={handleSubmit} class="space-y-6">
          <div>
            <label class="block text-zinc-400 text-xs uppercase mb-2 tracking-widest font-bold">Identity (Email)</label>
            <input
              type="email"
              value={email()}
              onInput={(e) => setEmail(e.currentTarget.value)}
              class="w-full bg-black border border-zinc-700 p-3 text-white focus:outline-none focus:border-red-600 transition-colors"
              placeholder="operator@oblivra.org"
              required
            />
          </div>

          <div>
            <label class="block text-zinc-400 text-xs uppercase mb-2 tracking-widest font-bold">Passphrase</label>
            <input
              type="password"
              value={password()}
              onInput={(e) => setPassword(e.currentTarget.value)}
              class="w-full bg-black border border-zinc-700 p-3 text-white focus:outline-none focus:border-red-600 transition-colors"
              placeholder="••••••••••••"
              required
            />
          </div>

          {error() && (
            <div class="bg-red-900/30 border border-red-800 p-3 text-red-500 text-xs uppercase font-bold text-center">
              ACCESS DENIED: {error()}
            </div>
          )}

          <button
            type="submit"
            disabled={loading()}
            class="w-full bg-white text-black font-black uppercase py-4 hover:bg-red-600 hover:text-white transition-all disabled:opacity-50 disabled:cursor-not-allowed group relative overflow-hidden"
          >
            <span class={loading() ? 'opacity-0' : 'relative z-10'}>Authorize Session</span>
            {loading() && (
              <div class="absolute inset-0 flex items-center justify-center">
                <div class="w-5 h-5 border-2 border-black border-t-transparent rounded-full animate-spin"></div>
              </div>
            )}
          </button>
        </form>

        <div class="mt-8 pt-6 border-t border-zinc-800 flex flex-col gap-4">
          <button 
            onClick={() => window.location.href = '/api/v1/auth/oidc/login'}
            class="w-full border border-zinc-700 text-zinc-500 text-xs uppercase font-bold py-2 hover:border-zinc-500 hover:text-zinc-300 transition-all"
          >
            Single Sign-On (OIDC)
          </button>
          <button 
            onClick={() => window.location.href = '/api/v1/auth/saml/login'}
            class="w-full border border-zinc-700 text-zinc-500 text-xs uppercase font-bold py-2 hover:border-zinc-500 hover:text-zinc-300 transition-all"
          >
            Federated Identity (SAML)
          </button>
        </div>
        
        <div class="mt-6 text-[10px] text-zinc-600 text-center uppercase tracking-[0.2em] font-medium leading-relaxed">
          Sovereign-Grade Encryption Active<br />
          Hardware Root-of-Trust Attestation: <span class="text-green-900">VERIFIED</span>
        </div>
      </div>
    </div>
  );
}
