<template>
  <div class="announce-text" @click="handleClick">
    <span
      v-for="(segment, index) in parsedSegments"
      :key="index"
      :style="{ color: segment.color }"
    >{{ segment.text }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  text: string;
  url?: string;
  clickable?: boolean;
}

interface TextSegment {
  text: string;
  color?: string;
}

const props = withDefaults(defineProps<Props>(), {
  clickable: false
});

const emit = defineEmits<{
  click: []
}>();

// Parse announce text with color codes (#RRGGBB)
const parsedSegments = computed(() => {
  const segments: TextSegment[] = [];
  // Convert \n and \r\n to actual line breaks
  const text = props.text.replace(/\\r\\n|\\n/g, '\n');

  // Regex to match #RRGGBB color codes followed by text
  const colorPattern = /#([0-9A-Fa-f]{6})(\S+)/g;
  let lastIndex = 0;
  let match;

  while ((match = colorPattern.exec(text)) !== null) {
    // Add text before the color code
    if (match.index > lastIndex) {
      segments.push({
        text: text.substring(lastIndex, match.index)
      });
    }

    // Add colored text
    segments.push({
      text: match[2],
      color: `#${match[1]}`
    });

    lastIndex = match.index + match[0].length;
  }

  // Add remaining text
  if (lastIndex < text.length) {
    segments.push({
      text: text.substring(lastIndex)
    });
  }

  return segments.length > 0 ? segments : [{ text }];
});

const handleClick = () => {
  if (props.clickable && props.url) {
    emit('click');
  }
};
</script>

<style scoped>
.announce-text {
  display: inline;
  word-wrap: break-word;
  white-space: pre-wrap;
}

.announce-text[clickable] {
  cursor: pointer;
  text-decoration: underline;
}
</style>
