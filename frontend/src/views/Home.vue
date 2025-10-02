<script setup lang="ts">
</script>

<template>
  <MyLayout>
    <template #bottom>
      <MyChart />
      <el-row :gutter="20" class="home-second-row" style="margin-left: 2px;">
        <el-col :span="12">
          <div class="home-box">
            <MyProfileCard />
          </div>
        </el-col>
        <el-col :span="12">
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
  max-width: 95%;
  margin-top: 30px;
  margin-left: 2px;
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
  height: 100%;
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
