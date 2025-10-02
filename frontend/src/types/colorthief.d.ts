// 颜色计算
declare module "colorthief" {
    export default class ColorThief {
        getColor(img: HTMLImageElement): [number, number, number];
        getPalette(img: HTMLImageElement, colorCount?: number): [number, number, number][];
    }
}
