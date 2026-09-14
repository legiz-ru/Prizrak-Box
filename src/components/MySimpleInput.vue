<template>
  <div class="input-wrapper">
    <input
        autocapitalize="off"
        autocomplete="off"
        spellcheck="false"
        @input="handleInput"
        :placeholder="placeholder"
        v-model="inputValue"
        class="custom-input"
    />
    <button
      v-if="inputValue"
      @click="clearInput"
      class="clear-button"
    >
      ✕
    </button>
  </div>
</template>

<script setup>
const props = defineProps({
  placeholder: String, // 占位符文本
  onInputChange: Function // 值变化时触发的回调方法
});

const inputValue = ref(''); // 双向绑定的输入值
const inputTimeout = ref(null); // 用于存储防抖的定时器

function handleInput(event) {
  const newValue = event.target.value;

  // 清除之前的定时器
  if (inputTimeout.value) {
    clearTimeout(inputTimeout.value);
  }

  // 设置新的定时器，延迟触发回调函数
  inputTimeout.value = setTimeout(() => {
    if (props.onInputChange) {
      props.onInputChange(newValue); // 调用回调函数
    }
  }, 500); // 设置防抖延迟时间，单位为毫秒
}

function clearInput() {
  inputValue.value = ''; // 清空输入框
  if (props.onInputChange) {
    props.onInputChange(''); // 通知父组件输入框已清空
  }
}
</script>

<style scoped>
.input-wrapper {
  position: relative;
  width: 100%;
}

.custom-input {
  width: 100%;
  /* Радиус, левый отступ и позиция кнопки очистки ниже — то, что раньше
     каждый из четырёх потребителей (Now/Ignore/ConnectionTab/LogTab)
     переопределял у себя одинаковыми `:deep()`-правилами; теперь это
     значения по умолчанию, а не то, что подставляется четыре раза. */
  padding: var(--px-space-2) 32px var(--px-space-2) var(--px-space-4);
  border: 1px solid var(--sub-card-border);
  border-radius: var(--px-r-pill);
  background-color: var(--left-nav-btn-bg);
  color: var(--text-color);
  font-size: var(--px-fs-body);
  box-sizing: border-box;
  outline: none;
  transition: border-color var(--px-dur) var(--px-ease), background-color var(--px-dur) var(--px-ease);
}

.custom-input:focus {
  background-color: var(--left-nav-btn-hover-bg);
}

.custom-input::placeholder {
  color: var(--placeholder-color);
}

.clear-button {
  position: absolute;
  top: 50%;
  right: 14px;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--text-color);
  opacity: .7;
  font-size: var(--px-fs-body);
  cursor: pointer;
  padding: 0;
  transition: opacity var(--px-dur) var(--px-ease);
}

.clear-button:hover {
  /* Было rgba(255,255,255,.8) буквально — не читалось на светлой теме. */
  opacity: 1;
}
</style>
