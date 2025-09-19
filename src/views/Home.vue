<script setup lang="ts">
</script>

<template>
  <MyLayout>
    <template #bottom>
      <MyChart />
      <el-row class="home-second-row" :gutter="20">
        <el-col :xs="24" :sm="24" :md="24" :lg="12">
          <div class="home-card">
            <MyProfileCard />
          </div>
        </el-col>
        <el-col :xs="24" :sm="24" :md="24" :lg="12">
          <div class="home-card">
            <MyTest />
          </div>
        </el-col>
      </el-row>
      <MyIp />
    </template>
  </MyLayout>
</template>

<style scoped>
.home-second-row {
  width: 100%;
  max-width: 95%;
  margin-top: 28px;
  margin-left: 2px;
}

.home-card {
  height: 100%;
  min-width: 0;
  display: flex;
}

.home-card :deep(.profile-card),
.home-card :deep(.profile-card-empty),
.home-card :deep(.t-card) {
  flex: 1;
  height: 100%;
}

.home-card :deep(.t-card) {
  width: 100%;
}
</style>
