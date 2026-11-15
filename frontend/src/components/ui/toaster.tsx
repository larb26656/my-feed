import { Toast } from "@base-ui/react/toast";

export const toastManager = Toast.createToastManager();

export function Toaster() {
	return (
		<Toast.Provider toastManager={toastManager}>
			<Toast.Portal>
				<Toast.Viewport className="fixed right-4 bottom-4 z-50 flex w-[min(24rem,calc(100vw-2rem))] flex-col gap-2 outline-none">
					<ToastList />
				</Toast.Viewport>
			</Toast.Portal>
		</Toast.Provider>
	);
}

function ToastList() {
	const { toasts } = Toast.useToastManager();
	return toasts.map((toast) => (
		<Toast.Root
			key={toast.id}
			toast={toast}
			className="rounded-lg border bg-card p-4 text-card-foreground shadow-lg data-[type=error]:border-destructive"
		>
			<Toast.Title className="font-medium" />
			<Toast.Description className="mt-1 text-sm text-muted-foreground" />
		</Toast.Root>
	));
}
