//
// модель взаимодействия нескольких goroutine посредсвом одного 
// разделяемого канала c буферизацией, реализована идея обыкновенной 
// СМО по схеме гибели и рождения
// для распараллеливания обработки используется моуль sync
//
package main

import(
	"fmt"
	"sync"
	"time"
	"errors" // errors.New()
	"math"
)

const(
	HYBRIDIZE = 20 // шанс выпадения оператора гибридизации
	WORKERS = 1000 // количество генетически различных форм
)

//
// MAIN DRIVER
//
func main() {
	fmt.Println("hello")
	
	//
	// UNBUFFERED CHANNEL
	//
	ch := make(chan int)
	
	// the proper way to send to an unbuffered channel
	// Notice that the send operation is wrapped in an
	// anonymous function invoked as a separate goroutine.
	go func() { ch<- 12 }()
	
	fmt.Println("chan recv:", <-ch)
	fmt.Println("done!")
	
	
	//
	// BUFFERED CHANNEL
	//
	chQ := make(chan int, 4)
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	// enqueue values
	fmt.Println("Enqueueing...")
	chQ<- 2
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	chQ<- 4
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	chQ<- 8
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	chQ<- 16
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	//chQ<- 32 // overflow, deadlock!
	//fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	
	// dequeue values
	fmt.Println("Dequeueing...")
	fmt.Println("1:", <-chQ)
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	fmt.Println("2:", <-chQ)
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	fmt.Println("3:", <-chQ)
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	fmt.Println("4:", <-chQ)
	fmt.Printf("cap: %d  len: %d\n", cap(chQ), len(chQ))
	//<-chQ // block, deadlock!	
	fmt.Println("done!")
	
	//
	// QUEUEING SYSTEM AND GA
	//
	// shake the generator!
	SRnd64(time.Now().Unix())
	
	var queue chan *Chromosome
	var wg sync.WaitGroup
	// подготовить и запустить алгоритм в параллельном режиме
	// при этом популяции будут обмениваться наиболее пригодными решениями
	// посредством разделяемой очереди или общего канала с буфирезацией
	// основная идея состоит в том, чтобы организовать общий мешок с двума отверстиями
	// в одно отверстие все попопуляции скидывают свои лучшие организмы(решения)
	// а через другое отверстие вытаскивыют имеющиеся в мешке организмы
	// за счёт рассинхронизации привносится элемент случайности, что позволяет ещё сильней размешивать(рандомизировать) решения
	// а это в свою очередь позволяет повысить вероятность отыскания оптимального или близкого к нему решения
	// развитие идеи и путь дальнейшего улучшейния лежит в сторону реализации какого-либо механизма
	// принудительного перемешивания материала. Что-то вроде барабана, в который с одной стороны производится загрузка
	// а с другой выгрузка, в нутри, тем временем, производится насколько это возможно тщательное и равномерное перемешивание 
	// (в идеале) до пулучения однородной смеси
	// 
	// необходимо задавать функцию или плотность распределения шансов выпадения генетических операторов	
	// из заданной функции распределения(или плотности) плучить значение шанса 
	// гибридизации(вероятность заявки на взаимодейтсвие популяций)
	// из полученного значения расчитать размер буфера как 
	// (среднее количесво занятых приборов + поток отказов обслуживания) * количесво популяций различных генетическх форм
	// (здесь это средняя загрузка + поток отказов) или n*lambda похоже
	// остановился на значении размера очереди как среднем значении загрузки * количесво популяций различных генетическх форм, его вроде хватает
	// значение среднего количесва занятых приборов использутеся для загрузки особоей, количетво загружаемых особей 
	// равно расчитанному среднему числу загруженных приборов
	
	// количество популяций различных генетических форм(или workers)
	workers := WORKERS
	
	// закон распределния вероятностей генетических операторов [0..100]
	hybridize := HYBRIDIZE
	// и другие...
	
	// расчитать размер очереди для взаимодействия популяций с целью получения гибридов
	// и среднее число загруженных приборов обслуживания
	// эти значения общие для всех популяций
	
	// поток заявок, здесь это запрос особи для гибридизации
	// или изъятие из общей очереди одного элемента
	lambda := float64(hybridize) // тест при лямбдя = 11
	// поток обслуживания, здесь это размер буфера, 
	// количество ячеек для особей
	mu := 1.0
	alpha := lambda / mu
	// количесво приборов обслуживания, здесь это ячейки т.к. одна ячейка обслуживает одну заявку mu = 1
	// тогда получим что количество обслуживающих приборов равно среднему числу завок lambda
	n := lambda 
	// расчитаем вероятность отсутствия заявок P0
	// для этого вычислим сумму
	sum := 0.0
	for k := 0.0; k < lambda; k++ {
		//p := math.Pow(alpha, k)
		//f := Factorial(k)
		sum += math.Pow(alpha, k)/Factorial(k)
		//fmt.Printf(" [%.0f]: p: %f  f: %f  p/f: %f\n", k, p, f, p/f)
	}
	P0 := 1 / sum
	Pn := math.Pow(alpha, n)/Factorial(n) * P0
	// поток отказов обслуживания
	refusal := lambda * Pn
	// вероятность обслуживания(или качество обслуживания)
	Q := 1 - Pn
	// среднее кличество занятых приборв
	averageLoad := lambda * Q
	// размер буфера 
	queueSize := (averageLoad /*+ refusal*/) * float64(workers)
	// создать буфер полученного размера
	queue = make(chan *Chromosome, int(queueSize))
	
	fmt.Printf("  lambda: %f\n", lambda)
	fmt.Printf("  mu: %f\n", mu)
	fmt.Printf("  alpha: %f\n", alpha)
	fmt.Printf("  n: %f\n", n)
	fmt.Printf("  Sum: %f\n", sum)
	fmt.Printf("  P0: %f\n", P0)
	fmt.Printf("  Pn: %f\n", Pn)
	fmt.Printf("  Refusal: %f\n", refusal)
	fmt.Printf("  Q: %f\n", Q)
	fmt.Printf("  Avg Load: %f\n", averageLoad)
	fmt.Printf("  QSz: %f\n", queueSize)
	//queue = make(chan *Chromosome, 10)
	var mc *MixerCylinder
	mc = NewMC(int(queueSize))
	mc.Start()
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id, quantity int, wg *sync.WaitGroup) {
			defer wg.Done()
			GA(id, quantity, &queue)
		}(i, workers, &wg)
	}
	wg.Wait()
	mc.Stop()
} // eof main



//
// MIXER CYLINDER
//
// представляет собой барабан-смеситель, который с одной стороны загружается черезе специальной загрузочное отверстие
// а с другой - разгружается через разгрузочное отверстие. При каждой загрузке производится перемешивание, таким образом
// материал в барабане постоянно перемешивается
// предусмотрены сигналы для управления установкой
type MixerCylinder struct {
	chargingHole chan struct{ 
		id int
		c *Chromosome
	}
	dischargingHole chan struct{
		c *Chromosome 
		err error
	}
	cylinder []*Chromosome
	//averageLoad // расчетная длина очереди
	signal chan struct{ stop bool }
}
// constructor
func NewMC(n int) *MixerCylinder {
	return new(MixerCylinder)
}
// основной итерфейс
// напоминают call у gen_server'a
// вообще следует почитать хорошо как устроен gen_server изнутри
// http://erlang.org/doc/man/gen_server.html
// http://erlang.org/doc/design_principles/gen_server_concepts.html
// https://github.com/erlang/otp/blob/master/lib/stdlib/src/gen_server.erl
// осуществляется отправка самомму собе и вызов соответствующих обработчиков
func (mc *MixerCylinder) Charge(id int, c *Chromosome) error {
	// здесь просходит простая постановка в очередь
	// и подаётся сигнал загрузки?наверное нет это автоматически происходит
	return nil
}
func (mc *MixerCylinder) Discharge(id int) (*Chromosome,error) {
	// подаётся сигнал на выгрузку, в канал управления
	// сервер помещает элемент из цилиндра в очередь
	// запросивший процесс встаёт в цикл ожидания ответа 
	// и ждёт появления сообщения в очереди
	// или таймаута
	return nil,nil
}
// интефейс управления
func (mc *MixerCylinder) Start() error {
	// создаётся сервер обрабатывающий три канала
	// канал загрузки
	// канал разгрузки
	// канал управления
	// защитный тайм-аут(длинный)
	// на каждый запрос производится вращение цилиндра
	// (перемешивание элементов слайса)
	return nil
}

func (mc *MixerCylinder) Stop() error {
	// отправляет сигнал остановки
	return nil
}
//
// GA MODEL
//

type Chromosome struct {
	a int
	b int
	c int
}
func NewChrome() *Chromosome {
	c := new(Chromosome)
	c.a = 0
	c.b = 0
	c.c = 0
	return c
}

type Population struct{
	id int
	individuals []*Chromosome
	queue *chan *Chromosome
	averageLoad int
	//
	requestsFlow float64
	serviceFlow float64
	requestsPerIter []float64
	servicesPerIter []float64
	//
	forms int
	hybridize int
	
}

func NewPop(popSz, id, workers int, queue *chan *Chromosome) *Population {
	p := new(Population)
	p.id = id
	p.individuals = make([]*Chromosome, popSz)
	p.queue = queue
	p.forms = workers
	p.averageLoad = int(cap(*queue)/workers)
	return p
}

func (p *Population) FoundingFathers() { 
	fmt.Printf("GA[%d]:FoundingFathers()\n", p.id)

	// необходимо задавать функцию или плотность распределения шансов выпадения генетических операторов
	
	// из заданной функции распределения(или плотности) плучить значение шанса 
	// гибридизации(вероятность заявки на взаимодейтсвие популяций)
	// из полученного значения расчитать размер буфера как 
	// (среднее количесво занятых приборов + поток отказов обслуживания) * количесво популяций различных генетическх форм
	// значение среднего количесва занятых приборов использутеся для загрузки особоей, количетво загружаемых особей 
	// равно расчитанному среднему числу загрузенных приборов
	
	// количество популяций различный генетических форм(или workers)
	//p.forms = 10
	
	// закон распределния вероятностей генетических операторов [0..100]
	p.hybridize = HYBRIDIZE
	
	/*
	// для ведущего у которого id==0
	// расчитать размер очереди для взаимодействия популяций для получения гибридов
	// и среднее число загруженных приборов обслуживания
	// эти значения общие для всех популяций
	if p.id == 0 {
		
		// это надо делать ДО запуска алгоритма, а иначе жуткий рассинхрон и половина времени работает в холостую
		// поток заявок, здесь это запрос особи для гибридизации
		// или изъятие из общей очереди одного элемента
		lambda := float64(p.hybridize * p.forms) // тест при лямбдя = 11
		// поток обслуживания, здесь это размер буфера, 
		// количество ячеек для особей
		mu := 1.0
		alpha := lambda / mu
		// количесво приборов обслуживания, здесь это ячейки т.к. одна ячейка обслуживает одну заявку mu = 1
		// тогда получим что количество обслуживающих приборов равно среднему числу завок lambda
		n := lambda 
		// расчитаем вероятность отсутствия заявок P0
		// для этого вычислим сумму
		sum := 0.0
		for k := 0.0; k < lambda; k++ {
			p := math.Pow(alpha, k)
			f := Factorial(k)
			sum += math.Pow(alpha, k)/Factorial(k)
			fmt.Printf(" [%.0f]: p: %f  f: %f  p/f: %f\n", k, p, f, p/f)
		}
		P0 := 1 / sum
		Pn := math.Pow(alpha, n)/Factorial(n) * P0
		// поток отказов обслуживания
		refusal := lambda * Pn
		// вероятность обслуживания
		Q := 1 - Pn
		// среднее кл-во занятых приборв
		averageLoad := lambda * Q
		// размер буфера как 
		// слишком много :) (среднее количесво занятых приборов+1 + поток отказов обслуживания+1) * количесво популяций различных генетическх форм
		// единицы добавлены чтобы число было округлено в бОльшу сторону, размер очереди должен быть заведо бОльшим чем средняя интенсивность
		//queueSize := float64(p.forms) * (averageLoad+1 + refusal+1)
		queueSize := averageLoad + refusal
		// создать буфера полученного размера
		(*p.queue) = make(chan *Chromosome, int(queueSize))
		// записать среденее число загруженных приборов
		//p.averageLoad = int(averageLoad)
		
		fmt.Printf("  lambda: %f\n", lambda)
		fmt.Printf("  mu: %f\n", mu)
		fmt.Printf("  alpha: %f\n", alpha)
		fmt.Printf("  n: %f\n", n)
		fmt.Printf("  Sum: %f\n", sum)
		fmt.Printf("  P0: %f\n", P0)
		fmt.Printf("  Pn: %f\n", Pn)
		fmt.Printf("  Refusal: %f\n", refusal)
		fmt.Printf("  Q: %f\n", Q)
		fmt.Printf("  Avg Load: %f\n", averageLoad)
		fmt.Printf("  QSz: %f\n", queueSize)
		
	}
	*/
	for i,_ := range p.individuals {
		//fmt.Printf("  initialize(%d)\n",i)
		p.individuals[i] = NewChrome()
	}	
}
func (p *Population) EvaluateFitness() { 
	//fmt.Printf("GA[%d]:EvaluateFitness()\n",p.id) 
	for i,_ := range p.individuals {
		i = i
		//fmt.Printf("  estimate(%d)\n",i)
	}
}
func (p *Population) SelectFittestSurvivors() { 
	//fmt.Printf("GA[%d]:SelectFittestSurvivors() Q: cap:%d  len: %d\n",p.id, cap(*(p.queue)), len(*(p.queue)))
	for i := (len(p.individuals) - (int(len(p.individuals)/4)+1)); i < len(p.individuals); i++ {
		//fmt.Printf("  select(%d)\n",i)
	}

	if p.id == 0 {
		fmt.Printf("   #####  Q cap: %d   len: %d   diff: %d\n", cap(*p.queue), len(*p.queue), cap(*p.queue)-len(*p.queue))	
	}

	c := NewChrome()
	c.a = p.id
	c.b = p.id
	c.c = p.id
	p.enqueue(c)	
}
func (p *Population) Recombine() { 
	//fmt.Printf("GA[%d]:Recombine()\n", p.id)
	//defer func() {
		//
		// To be able to recover from an unwinding panic sequence, 
		// the code must make a deferred call to the recover function.
		//
	//	if r := recover(); r != nil {
	//		fmt.Println("[W] Reader FAULT")
	//	}
	//}()
	
	reqCounter := 0
	for i,_ := range p.individuals {
		i = i
		//fmt.Printf("  recombine(%d)\n",i)
		// разыграть шанс оператора гибридизации - скреFщивания генетически различающихся форм
		chance := RndBetweenU(0,100)
		//TODO: шанс выпадения для обмена должен сильно зависеть от размера популяции!! это важно!
		// елси выпал шанс оператора и буфер гибридизации не пуст, тогда и только тогда произвести гибридизацию
		// достать особь из накопителя и произвести скрещивание
		if chance <= p.hybridize {
			reqCounter++
			p.dequeue()	
		}
	}
	if reqCounter > 0 {
		p.requestsPerIter = append(p.requestsPerIter, float64(reqCounter))
	}
}

func (p *Population) enqueue(c *Chromosome) error {
	
	// отказ если в очереди уже нет мест
	if len( *(p.queue) ) < cap( *(p.queue) ) {
		fmt.Printf("[%d]  enqueue fittest\n", p.id)
		go func(c *Chromosome) { 
			//defer func() {
				//
				// To be able to recover from an unwinding panic sequence, 
				// the code must make a deferred call to the recover function.
				//
			//	if r := recover(); r != nil {
			//		fmt.Println("[W] Reader FAULT")
			//	}
			//}()
			fmt.Println("[",p.id, "] ENQ chrome:", c)
			srvCounter := 0
			for i := 0; i < p.averageLoad-1; i++ {
				//fmt.Printf("[%d]  before:  Q cap: %d  len: %d  diff: %d\n",p.id, cap(*(p.queue)), len(*(p.queue)), cap(*(p.queue))-len(*(p.queue)))
				*(p.queue)<- c
				//fmt.Printf("[%d]  enqueue:  Q cap: %d  len: %d  diff: %d\n",p.id, cap(*(p.queue)), len(*(p.queue)), cap(*(p.queue))-len(*(p.queue)))
				srvCounter++
			}
			p.servicesPerIter = append(p.servicesPerIter, float64(srvCounter))
		}(c)
		return nil
	} else {
		return errors.New("QUEUEFULL")
	}
}
func (p *Population) dequeue() (*Chromosome,error) {
	// отказ если очередь пуста
	if len( *(p.queue) ) > 0 {
		//fmt.Printf("[%d]    hybridize -> before  Q cap: %d  len: %d\n",p.id, cap( *(p.queue) ), len( *(p.queue) ) )	
		c := <-*(p.queue)
		//fmt.Printf("[%d]    hybridize -> dequeue:  Q cap: %d  len: %d\n",p.id, cap( *(p.queue) ), len( *(p.queue) ) )
		fmt.Println("[", p.id, "] DEQ chrome:", c)
		return c,nil
	} else {
		return nil,errors.New("EMPTYQUEUE")
	}
}
func (p *Population) Result() { 
	fmt.Printf("GA[%d]:Result()\n",p.id)
	fmt.Printf("[%d]  Avg requests per interation: %f\n",p.id, Mean(p.requestsPerIter))
	fmt.Printf("[%d]  Avg services per interation: %f\n",p.id, Mean(p.servicesPerIter))
	//fmt.Println(p.requestsPerIter)
}
func (p *Population) Done() bool { return false }

func GA(id, quantity int,  queue *chan *Chromosome) {
	// 
	// Genetic Algorithm
	//
	// создать популяцию численности N
	populationSize := 100
	pop := NewPop(populationSize, id, quantity, queue)
	// сформировать исходную популяцию или начальное приближение
	pop.FoundingFathers()
	// максимальное число итераций
	var niterations = 3000
	for i := 0; i < niterations && !pop.Done(); i++ {
		fmt.Printf("   GENERATION[%d]\n", i)
		// Estimate population fitnesses
		pop.EvaluateFitness()
		// There will be selection of the fittest
		pop.SelectFittestSurvivors()
		// Spawn new generation from the most fittest
		pop.Recombine()
	}
	// Get result
	pop.Result()
	fmt.Println()
}



var (
	gRndSeed uint32 = 1 // последнее случайное число
	was int = 0 // была ли вычислена пара чисел
	r float64= 0 // предыдущее число
)
// Начиная с некоторого целого числа x0 =/= 0, задаваемого при помощи фукнции SRnd(),
// при каждом вызове функции Rnd() происходит вычисление нового псевдослучайного 
// числа на основе предыдущего.
func SRnd64(seed int64) {
	SRnd(uint32(seed))
}

func SRnd(seed uint32) {
	if seed == uint32(0) {
		gRndSeed = uint32(1)
	} else {
		gRndSeed = seed
	}
}
// Метод генерации случайных чисел основанный на эффекет переполнения 32-разрядных целых чисел
// возвращает равномерно распределённое случайное число
func RndU() uint32 {
	gRndSeed = gRndSeed * uint32(1664525) + uint32(1013904223)
	return gRndSeed
}
// генерировать челое число из диапазона
// с типами надо подумать...
func RndBetweenU(bottom, top int) (result int) {
	// формула генерации случайных чисел по заданному диапазону
	// где bottom - минимальное число из желаемого диапазона
	// top - верхнаяя граница, ширина выборки
	rnd := int(RndU())
	div := rnd % top
	diff := top - div
	if diff > bottom {
		result = bottom + div
	} else {
		result = div
	}
	return
}


func Factorial(n float64) float64 {
	if n == float64(0) {
		return float64(1)
	} else if n == float64(1) {
		return float64(1)
	} else {
		//FIXME: return math.Sqrt2 * math.SqrtPi * float64(n) * math.Pow(float64(n)/math.E,float64(n))
		return math.Sqrt(2.0 * math.Pi * n) * math.Pow(n/math.E, n)
	}
}

// Mean calculates the arithmetic mean with the recurrence relation
func Mean(data []float64) (mean float64) {
	Len := len(data)
	for i := 0; i < Len; i++ {
		mean += (data[i] - mean) / float64(i+1)
	}
	return
}