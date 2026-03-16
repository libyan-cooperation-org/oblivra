import { Component, createSignal, Show, onMount } from 'solid-js';
import { useApp } from '@core/store';
import { HostTree } from './HostTree';
import { QuickConnect } from './QuickConnect';
import { AddHostModal } from './AddHostModal';
import { database } from '../../../wailsjs/go/models';
import { Delete, ListHosts } from '../../../wailsjs/go/services/HostService';

export const Sidebar: Component = () => {
  const [state, actions] = useApp();
  const [activeTab, setActiveTab] = createSignal<'hosts' | 'snippets' | 'history'>('hosts');
  const [searchQuery, setSearchQuery] = createSignal('');
  const [showAddHost, setShowAddHost] = createSignal(false);
  const [editingHost, setEditingHost] = createSignal<database.Host | undefined>(undefined);

  onMount(async () => {
    try {
      const hosts = await ListHosts();
      if (hosts) {
        const mappedHosts = hosts.map((h: any) => ({
          ...h,
          authMethod: h.auth_method || 'password',
          isFavorite: h.is_favorite || false,
          connectionCount: h.connection_count || 0
        }));
        actions.setHosts(mappedHosts);
      }
    } catch (err) {
      console.error("Failed to load hosts on startup:", err);
    }
  });

  const handleEditHost = (hostId: string) => {
    const host = state.hosts.find(h => h.id === hostId);
    if (host) {
      setEditingHost(host as any);
      setShowAddHost(true);
    }
  };

  const handleDeleteHost = async (hostId: string) => {
    if (confirm('Are you sure you want to delete this host?')) {
      try {
        await Delete(hostId);
        actions.removeHost(hostId);
      } catch (err) {
        alert('Failed to delete host: ' + (err as Error).message);
      }
    }
  };

  return (
    <aside class="sidebar">
      {/* Quick Connect */}
      <QuickConnect />

      {/* Directory Header with Top Actions */}
      <div class="sidebar-header-row">
        <div class="sidebar-title">Directory</div>
        <div class="sidebar-actions-top">
          <button class="sidebar-icon-btn" onClick={() => setShowAddHost(true)} title="Add Host">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
          </button>
          <button class="sidebar-icon-btn" onClick={() => { }} title="Import Configuration">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
          </button>
        </div>
      </div>

      {/* Search */}
      <div class="sidebar-search-container">
        <span class="search-icon">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
        </span>
        <input
          type="text"
          placeholder={`Search ${activeTab()}...`}
          value={searchQuery()}
          onInput={(e) => setSearchQuery(e.currentTarget.value)}
        />
      </div>

      {/* MacOS-style Segmented Control */}
      <div class="ob-segmented-control">
        <button
          class={activeTab() === 'hosts' ? 'active' : ''}
          onClick={() => setActiveTab('hosts')}
        >
          Hosts
        </button>
        <button
          class={activeTab() === 'snippets' ? 'active' : ''}
          onClick={() => setActiveTab('snippets')}
        >
          Snippets
        </button>
        <button
          class={activeTab() === 'history' ? 'active' : ''}
          onClick={() => setActiveTab('history')}
        >
          History
        </button>
      </div>

      {/* Content */}
      <div class="sidebar-content">
        <Show when={activeTab() === 'hosts'}>
          <HostTree
            searchQuery={searchQuery()}
            onEditHost={handleEditHost}
            onDeleteHost={handleDeleteHost}
          />
        </Show>

        <Show when={activeTab() === 'snippets'}>
          <div class="sidebar-empty-state">
            <div class="sidebar-empty-icon">📝</div>
            <div class="sidebar-empty-title">No snippets created</div>
            <div class="sidebar-empty-desc">Create reusable command snippets to execute across multiple hosts instantly.</div>
            <button class="ob-btn ob-btn-ghost ob-btn-sm" style="margin-top: 8px;">
              + Create Snippet
            </button>
          </div>
        </Show>

        <Show when={activeTab() === 'history'}>
          <div class="sidebar-empty-state">
            <div class="sidebar-empty-icon">🕒</div>
            <div class="sidebar-empty-title">History is empty</div>
            <div class="sidebar-empty-desc">Your recent SSH sessions and playbook executions will appear here.</div>
          </div>
        </Show>
      </div>

      <Show when={showAddHost()}>
        <AddHostModal
          host={editingHost()}
          onClose={() => {
            setShowAddHost(false);
            setEditingHost(undefined);
          }}
        />
      </Show>
    </aside>
  );
};