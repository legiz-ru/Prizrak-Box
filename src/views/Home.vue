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
  display: grid;
  gap: 20px;
  grid-template-columns: minmax(320px, 0.9fr) minmax(320px, 1.4fr);
  grid-auto-rows: 1fr;
}

.home-profile,
.home-test {
  min-width: 0;
  display: flex;
}

.home-profile :deep(.profile-card),
.home-profile :deep(.profile-card-empty),
.home-test :deep(.t-card) {
  flex: 1;
  height: 100%;
}

.home-test :deep(.t-card) {
  width: 100%;
}

@media (max-width: 1280px) {
  .home-second-row {
    grid-template-columns: minmax(280px, 1fr);
  }
}
</style>
