<script setup lang="ts">
</script>

<template>
  <MyLayout>
    <template #bottom>
      <MyChart />
      <el-row class="home-second-row" :gutter="20" align="stretch">
        <el-col :span="12" :xs="24" :sm="24" class="home-card">
          <MyProfileCard />
        </el-col>
        <el-col :span="12" :xs="24" :sm="24" class="home-card">
          <MyTest />
        </el-col>
      </el-row>
      <MyIp />
    </template>
  </MyLayout>
</template>

<style scoped>
.home-second-row {
  width: 95%;
  max-width: 95%;
  margin-top: 28px;
  margin-left: 2px;
}

.home-card {
  display: flex;
  height: 100%;
  min-width: 0;
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
