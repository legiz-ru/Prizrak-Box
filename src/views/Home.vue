<script setup lang="ts">
</script>

<template>
  <MyLayout>
    <template #bottom>
      <MyChart />
      <div class="home-second-row">
        <div class="home-profile">
          <MyProfileCard />
        </div>
        <div class="home-test">
          <MyTest />
        </div>
      </div>
      <MyIp />
    </template>
  </MyLayout>
</template>

<style scoped>
.home-second-row {
  width: calc(95% - 12px);
  margin-left: 12px;
  margin-top: 28px;
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
}

.home-profile {
  flex: 1 1 320px;
  min-width: 280px;
}

.home-test {
  flex: 2 1 480px;
  min-width: 320px;
}
</style>