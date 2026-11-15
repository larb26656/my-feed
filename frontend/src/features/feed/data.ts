export type Entry = {
	id: number;
	source: string;
	sourceId: string;
	initials: string;
	color: string;
	time: string;
	title: string;
	description: string;
	category: string;
	read: string;
	artwork: string;
};
export const entries: Entry[] = [
	{
		id: 1,
		source: "The New York Times",
		sourceId: "the-new-york-times",
		initials: "T",
		color: "ink",
		time: "2 ชั่วโมงที่แล้ว",
		title: "The little things that make a city feel like home",
		description: "เรื่องราวของมุมเล็ก ๆ ร้านประจำ และผู้คนที่ทำให้เมืองใหญ่รู้สึกใกล้ชิดกว่าที่เคย",
		category: "CULTURE",
		read: "อ่าน 6 นาที",
		artwork: "city",
	},
	{
		id: 2,
		source: "The Creative Independent",
		sourceId: "the-creative-independent",
		initials: "C",
		color: "violet",
		time: "4 ชั่วโมงที่แล้ว",
		title: "On making space for the ideas that matter",
		description:
			"ชวนกลับมามองกระบวนการสร้างสรรค์ และให้เวลากับไอเดียที่ยังไม่จำเป็นต้องรีบเติบโต",
		category: "CREATIVITY",
		read: "อ่าน 8 นาที",
		artwork: "creative",
	},
	{
		id: 3,
		source: "The Verge",
		sourceId: "the-verge",
		initials: "V",
		color: "orange",
		time: "เมื่อวาน",
		title: "A more human way to think about technology",
		description: "เมื่อเทคโนโลยีที่ดีไม่ใช่แค่ฉลาดขึ้น แต่ช่วยให้เรามีเวลาให้สิ่งสำคัญมากขึ้น",
		category: "TECHNOLOGY",
		read: "อ่าน 5 นาที",
		artwork: "tech",
	},
	{
		id: 4,
		source: "Kinfolk",
		sourceId: "kinfolk",
		initials: "K",
		color: "sand",
		time: "เมื่อวาน",
		title: "A slower kind of Sunday",
		description:
			"บันทึกเล็ก ๆ ว่าด้วยการพักผ่อน กาแฟแก้วโปรด และความสุขของการไม่ต้องรีบไปไหน",
		category: "LIFESTYLE",
		read: "อ่าน 4 นาที",
		artwork: "sunday",
	},
];
export const filters = ["ทั้งหมด", "ยังไม่อ่าน"] as const;
export type FeedFilter = (typeof filters)[number];
