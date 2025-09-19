<script setup lang="ts">
</script>

<template>
  <MyLayout>
    <template #bottom>
      <MyChart />
      <el-row class="home-second-row" :gutter="20" align="stretch" style="margin-left: 2px;">
        <el-col :span="12" :xs="24" :sm="24">
          <div class="home-box">
            <MyProfileCard />
          </div>
        </el-col>
        <el-col :span="12" :xs="24" :sm="24">
          <div class="home-box">
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
  width: 95%;
  max-width: 95%;
  margin-top: 30px;
}

.home-box {
  width: 100%;
  height: 100%;
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
</style>
