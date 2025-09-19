<script setup lang="ts">
</script>

<template>
  <MyLayout>
    <template #bottom>
      <MyChart />
      <div class="home-second-row">
        <div class="home-second-col">
          <div class="home-box">
            <MyProfileCard />
          </div>
        </div>
        <div class="home-second-col">
          <div class="home-box">
            <MyTest />
          </div>
        </div>
      </div>
      <MyIp />
    </template>
  </MyLayout>
</template>

<style scoped>

.home-second-row {
  width: 95%;
  max-width: 95%;
  margin-top: 30px;
  margin-left: 2px;
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
}

.home-second-col {
  flex: 1 1 calc(50% - 10px);
  min-width: 320px;
  display: flex;
}

.home-box {
  width: 100%;
  padding: 10px;
  border-radius: 8px;
  text-align: left;
  box-shadow: var(--right-box-shadow);
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.home-box :deep(.profile-card),
.home-box :deep(.profile-card-empty),
.home-box :deep(.t-card) {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
}

@media (max-width: 1024px) {
  .home-second-col {
    flex: 1 1 100%;
  }
}
</style>
