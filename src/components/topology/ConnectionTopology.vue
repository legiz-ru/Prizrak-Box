<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue';
import * as echarts from 'echarts';
import type { ECharts } from 'echarts';
import 'country-flag-emoji-polyfill';

interface ConnectionData {
  id: string;
  metadata: {
    host?: string;
    destinationIP: string;
    destinationPort: number;
    sourceIP?: string;
    process?: string;
    type: string;
  };
  start: string;
  download: number;
  upload: number;
  rule: string;
  rulePayload?: string;
  chains: string[];
}

const props = defineProps<{
  connections: ConnectionData[];
}>();

const chartContainer = ref<HTMLElement>();
const chart = ref<ECharts>();
const isPaused = ref(false);

// Sankey diagram data structure
interface SankeyNode {
  name: string;
}

interface SankeyLink {
  source: string;
  target: string;
  value: number;
}

// Helper: Extract rule type from rule string
const getRuleType = (rule: string): string => {
  if (!rule) return 'Direct';

  // Match different rule formats
  if (rule === 'Match') return 'Match';
  if (rule.startsWith('RuleSet:')) return 'RuleSet';
  if (rule.startsWith('DOMAIN')) return 'DOMAIN';
  if (rule.startsWith('IP-CIDR')) return 'IP-CIDR';
  if (rule.startsWith('GEOIP')) return 'GEOIP';
  if (rule.startsWith('PROCESS')) return 'PROCESS';
  if (rule.startsWith('AND')) return 'AND';
  if (rule.startsWith('OR')) return 'OR';
  if (rule.startsWith('NOT')) return 'NOT';

  // Extract first part before comma or colon
  const match = rule.match(/^([A-Z-]+)[,:]/);
  return match ? match[1] : rule;
};

// Helper: Add country flag emoji to proxy name
const addCountryFlag = (proxyName: string): string => {
  if (!proxyName || proxyName === 'Direct' || proxyName === 'DIRECT') return proxyName;

  // Country code mapping
  const countryMap: Record<string, string> = {
    'US': '🇺🇸', 'UK': '🇬🇧', 'HK': '🇭🇰', 'JP': '🇯🇵', 'SG': '🇸🇬',
    'KR': '🇰🇷', 'TW': '🇹🇼', 'CN': '🇨🇳', 'DE': '🇩🇪', 'FR': '🇫🇷',
    'CA': '🇨🇦', 'AU': '🇦🇺', 'RU': '🇷🇺', 'IN': '🇮🇳', 'BR': '🇧🇷',
    'NL': '🇳🇱', 'SE': '🇸🇪', 'CH': '🇨🇭', 'IT': '🇮🇹', 'ES': '🇪🇸',
  };

  // Try to find country code in proxy name
  for (const [code, flag] of Object.entries(countryMap)) {
    // Check if proxy name contains country code
    const regex = new RegExp(`\\b${code}\\b`, 'i');
    if (regex.test(proxyName) && !proxyName.includes(flag)) {
      return `${flag} ${proxyName}`;
    }
  }

  return proxyName;
};

// Helper: Format rule with payload
const formatRule = (rule: string, rulePayload?: string): string => {
  if (!rule) return 'Direct';

  const ruleType = getRuleType(rule);

  if (rulePayload) {
    // Truncate very long payloads
    const maxPayloadLength = 40;
    const payload = rulePayload.length > maxPayloadLength
      ? rulePayload.substring(0, maxPayloadLength) + '...'
      : rulePayload;
    return `${ruleType}: ${payload}`;
  }

  return ruleType;
};

// Process connections into Sankey data
const sankeyData = computed(() => {
  const nodes: SankeyNode[] = [];
  const links: SankeyLink[] = [];
  const nodeSet = new Set<string>();
  const linkMap = new Map<string, number>();

  // Store which nodes are proxies (to show destinations in tooltip)
  const proxyNodes = new Set<string>();
  // Store destinations for each proxy (for tooltip)
  const proxyDestinations = new Map<string, Set<string>>();

  if (!props.connections || props.connections.length === 0) {
    return { nodes: [], links: [], proxyNodes, proxyDestinations };
  }

  props.connections.forEach((conn) => {
    // Build the complete flow chain
    const flowChain: string[] = [];

    // Layer 1: Process (or connection type if no process)
    const process = conn.metadata.process || conn.metadata.type || 'Unknown';
    flowChain.push(process);

    // Layer 2: Rule with payload
    const rule = formatRule(conn.rule, conn.rulePayload);
    flowChain.push(rule);

    // Layer 3-N: All proxies from chains (reverse order - chains go from exit to entry)
    if (conn.chains && conn.chains.length > 0) {
      // Reverse chains: they come as [exit, middle, entry] but we want [entry, middle, exit]
      const reversedChains = [...conn.chains].reverse();
      reversedChains.forEach((proxy) => {
        const proxyWithFlag = addCountryFlag(proxy);
        flowChain.push(proxyWithFlag);
        proxyNodes.add(proxyWithFlag);
      });
    } else {
      flowChain.push('Direct');
      proxyNodes.add('Direct');
    }

    // Store destination for the last proxy (for tooltip, not as separate layer)
    const destination = conn.metadata.host || conn.metadata.destinationIP || 'Unknown';
    const lastProxy = flowChain[flowChain.length - 1];
    if (!proxyDestinations.has(lastProxy)) {
      proxyDestinations.set(lastProxy, new Set());
    }
    proxyDestinations.get(lastProxy)!.add(destination);

    // Add all nodes from the chain
    flowChain.forEach(node => {
      if (!nodeSet.has(node)) {
        nodeSet.add(node);
        nodes.push({ name: node });
      }
    });

    // Create links between consecutive nodes in the chain
    for (let i = 0; i < flowChain.length - 1; i++) {
      const source = flowChain[i];
      const target = flowChain[i + 1];
      const linkKey = `${source}->${target}`;
      linkMap.set(linkKey, (linkMap.get(linkKey) || 0) + 1);
    }
  });

  // Convert link map to array with logarithmic scaling
  linkMap.forEach((value, key) => {
    const [source, target] = key.split('->');
    // Apply logarithmic scaling for better visibility
    const scaledValue = Math.log10(value + 1) * 10;
    links.push({ source, target, value: scaledValue });
  });

  // Sort nodes alphabetically
  nodes.sort((a, b) => a.name.localeCompare(b.name));

  return { nodes, links, proxyNodes, proxyDestinations };
});

// Get theme colors from CSS variables
const getThemeColors = () => {
  if (typeof window === 'undefined') return {
    text: '#333',
    background: '#fff',
    border: '#aaa',
  };

  const root = document.documentElement;
  const styles = getComputedStyle(root);

  return {
    text: styles.getPropertyValue('--text-color') || '#333',
    background: styles.getPropertyValue('--left-bg-color') || '#fff',
    border: styles.getPropertyValue('--sub-card-border') || '#aaa',
  };
};

// ECharts options
const chartOptions = computed(() => {
  const colors = getThemeColors();

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      backgroundColor: colors.background,
      borderColor: colors.border,
      textStyle: {
        color: colors.text,
        fontFamily: "'Twemoji', 'Nunito', 'Microsoft YaHei', sans-serif",
      },
      formatter: (params: any) => {
        if (params.dataType === 'edge') {
          const actualCount = Math.round(Math.pow(10, params.data.value / 10) - 1);
          return `${params.data.source} → ${params.data.target}<br/>Connections: ${actualCount}`;
        }

        // For proxy nodes, show only destinations list
        const nodeName = params.name;
        const destinations = sankeyData.value.proxyDestinations.get(nodeName);

        if (destinations && destinations.size > 0) {
          const destArray = Array.from(destinations);
          const maxShow = 10;
          let tooltip = '<strong>Destinations:</strong><br/>';

          destArray.slice(0, maxShow).forEach(dest => {
            tooltip += `• ${dest}<br/>`;
          });

          if (destArray.length > maxShow) {
            tooltip += `<em>... and ${destArray.length - maxShow} more</em>`;
          }

          return tooltip;
        }

        return nodeName;
      },
    },
    series: [
      {
        type: 'sankey',
        layout: 'none',
        orient: 'horizontal',
        top: 10,
        bottom: 10,
        left: 5,
        right: 80,
        nodeWidth: 25,
        nodeGap: 12,
        emphasis: {
          focus: 'adjacency',
        },
        lineStyle: {
          color: 'gradient',
          curveness: 0.5,
          opacity: 0.4,
        },
        itemStyle: {
          borderWidth: 1,
          borderColor: colors.border,
        },
        label: {
          color: colors.text,
          fontSize: 12,
          fontWeight: 500,
          fontFamily: "'Twemoji', 'Nunito', 'Microsoft YaHei', sans-serif",
          position: 'right',
          formatter: (params: any) => {
            const maxLength = 35;
            const name = params.name;
            return name.length > maxLength ? name.substring(0, maxLength) + '...' : name;
          },
        },
        data: sankeyData.value.nodes,
        links: sankeyData.value.links,
      },
    ],
  };
});

// Initialize chart
const initChart = () => {
  if (!chartContainer.value) return;

  if (chart.value) {
    chart.value.dispose();
  }

  // Use SVG renderer for better emoji support
  chart.value = echarts.init(chartContainer.value, null, {
    renderer: 'svg',
  });
  updateChart();
};

// Update chart with new data
const updateChart = () => {
  if (!chart.value || isPaused.value) return;

  chart.value.setOption(chartOptions.value, true);
};

// Handle window resize
const handleResize = () => {
  if (chart.value) {
    chart.value.resize();
  }
};

// Toggle pause/resume
const togglePause = () => {
  isPaused.value = !isPaused.value;
  if (!isPaused.value) {
    updateChart();
  }
};

// Watch for connection changes
watch(
  () => props.connections,
  () => {
    if (!isPaused.value) {
      nextTick(() => {
        updateChart();
      });
    }
  },
  { deep: true }
);

watch(
  () => sankeyData.value.nodes.length,
  (nodeCount) => {
    if (nodeCount > 0) {
      nextTick(() => {
        if (!chart.value) {
          initChart();
        } else {
          updateChart();
        }
      });
    }
  }
);

// Lifecycle hooks
onMounted(() => {
  nextTick(() => {
    initChart();
  });
  window.addEventListener('resize', handleResize);
});

onUnmounted(() => {
  window.removeEventListener('resize', handleResize);
  if (chart.value) {
    chart.value.dispose();
  }
});
</script>

<template>
  <div class="topology-container">
    <div class="topology-header">
      <div class="topology-title">
        {{ $t('connections.topology') }}
      </div>
      <div class="topology-controls">
        <el-button
          @click="togglePause"
          circle
          size="small"
          :title="isPaused ? $t('connections.resume') : $t('connections.pause')"
        >
          <icon-mdi-play v-if="isPaused" />
          <icon-mdi-pause v-else />
        </el-button>
      </div>
    </div>
    <div
      v-if="sankeyData.nodes.length > 0"
      ref="chartContainer"
      class="chart-container"
    ></div>
    <div v-else class="no-data">
      <el-empty :description="$t('connections.noData')" />
    </div>
  </div>
</template>

<style scoped>
.topology-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: transparent;
  overflow: hidden;
  min-height: calc(100vh - 220px);
}

.topology-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: var(--left-bg-color);
  border-radius: 8px 8px 0 0;
  flex-shrink: 0;
}

.topology-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.topology-controls {
  display: flex;
  gap: 8px;
}

.chart-container {
  flex: 1;
  width: 100%;
  height: 100%;
  min-height: 0;
  background: var(--left-bg-color);
  border-radius: 0 0 8px 8px;
}

.no-data {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 500px;
  background: var(--left-bg-color);
  border-radius: 0 0 8px 8px;
}
</style>
