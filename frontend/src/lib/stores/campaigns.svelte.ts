import { writable, derived } from 'svelte/store';

export interface CampaignCluster {
    cluster_id: string;
    entities: string[];
    tactics: string[];
    tactic_hits: Record<string, number>;
    edge_count: number;
    first_seen: string;
    last_seen: string;
}

function createCampaignStore() {
    const { subscribe, set, update } = writable<CampaignCluster[]>([]);

    return {
        subscribe,
        setCampaigns: (campaigns: CampaignCluster[]) => set(campaigns),
        updateCampaign: (campaign: CampaignCluster) => update(list => {
            const index = list.findIndex(c => c.cluster_id === campaign.cluster_id);
            if (index !== -1) {
                list[index] = campaign;
                return [...list];
            }
            return [...list, campaign];
        }),
        clear: () => set([])
    };
}

export const campaignStore = createCampaignStore();
